package socket

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/jjeffcaii/reactor-go"
	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/pkg/errors"
	"go.uber.org/atomic"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/internal/bytesconv"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/fragmentation"
	"github.com/rsocket/rsocket-go/internal/misc"
	"github.com/rsocket/rsocket-go/internal/queue"
	"github.com/rsocket/rsocket-go/lease"
	"github.com/rsocket/rsocket-go/logger"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/rsocket/rsocket-go/rx/mono"
)

const _outChanSize = 64
const _minRequestSchedulerSize = 1000

var errSocketClosed = errors.New("rsocket: socket closed already")
var errRequestFailed = errors.New("rsocket: send request failed")

var (
	unsupportedRequestStream   = []byte("Request-Stream not implemented.")
	unsupportedRequestResponse = []byte("Request-Response not implemented.")
	unsupportedRequestChannel  = []byte("Request-Channel not implemented.")
)

func mustExecute(sc scheduler.Scheduler, handler func()) {
	if err := sc.Worker().Do(handler); err == nil {
		return
	}
	go handler()
}

// DuplexConnection represents a socket of RSocket which can be a requester or a responder.
type DuplexConnection struct {
	ctx            context.Context
	reqSche        scheduler.Scheduler
	resSche        scheduler.Scheduler
	destroyReqSche bool
	locker         sync.RWMutex
	counter        *core.TrafficCounter
	tp             *transport.Transport
	sndQueue       chan core.WriteableFrame
	sndBacklog     []core.WriteableFrame
	responder      Responder
	messages       sync.Map // key=streamID, value=callback
	sids           StreamID
	mtu            int
	fragments      sync.Map // key=streamID, value=Joiner
	writeDone      chan struct{}
	keepaliver     *Keepaliver
	cond           sync.Cond
	e              error
	leases         lease.Factory
	closed         *atomic.Bool
	ready          *atomic.Bool
}

// SetError sets error for current socket.
func (dc *DuplexConnection) SetError(err error) {
	dc.locker.Lock()
	dc.e = err
	dc.locker.Unlock()
}

// GetError get the error set.
func (dc *DuplexConnection) GetError() (err error) {
	dc.locker.RLock()
	err = dc.e
	dc.locker.RUnlock()
	return
}

func (dc *DuplexConnection) nextStreamID() (sid uint32) {
	var firstLap bool
	for {
		// There's no required to check StreamID conflicts.
		sid, firstLap = dc.sids.Next()
		if firstLap {
			return
		}
		_, ok := dc.messages.Load(sid)
		if !ok {
			return
		}
	}
}

// Close close current socket.
func (dc *DuplexConnection) Close() error {
	if !dc.closed.CAS(false, true) {
		return nil
	}

	if dc.keepaliver != nil {
		dc.keepaliver.Stop()
	}
	close(dc.sndQueue)

	dc.cond.L.Lock()
	dc.cond.Broadcast()
	dc.cond.L.Unlock()

	var writeDone chan struct{}

	dc.locker.RLock()
	writeDone = dc.writeDone
	dc.locker.RUnlock()

	// wait for write loop end
	if writeDone != nil {
		<-dc.writeDone
	}

	dc.destroyTransport()

	err := dc.GetError()
	if err == nil {
		dc.destroyHandler(errSocketClosed)
	} else {
		dc.destroyHandler(err)
	}

	dc.destroySndQueue()
	dc.destroySndBacklog()
	dc.destroyFragment()

	if dc.destroyReqSche {
		_ = dc.reqSche.Close()
	}

	return err
}

func (dc *DuplexConnection) destroyTransport() {
	dc.locker.Lock()
	defer dc.locker.Unlock()
	if tp := dc.tp; tp != nil {
		if dc.e == nil {
			dc.e = tp.Close()
		} else {
			_ = tp.Close()
		}
	}
}

func (dc *DuplexConnection) destroyHandler(err error) {
	dc.messages.Range(func(_, value interface{}) bool {
		cb := value.(callback)
		cb.stopWithError(err)
		return true
	})
}

func (dc *DuplexConnection) destroyFragment() {
	dc.fragments.Range(func(_, i interface{}) bool {
		common.TryRelease(i)
		return true
	})
}

func (dc *DuplexConnection) destroySndQueue() {
	for next := range dc.sndQueue {
		next.Done()
	}
}

func (dc *DuplexConnection) destroySndBacklog() {
	n := len(dc.sndBacklog)
	for i := 0; i < n; i++ {
		dc.sndBacklog[i].Done()
		dc.sndBacklog[i] = nil
	}
	dc.sndBacklog = nil
}

// FireAndForget start a request of FireAndForget.
func (dc *DuplexConnection) FireAndForget(req payload.Payload) {
	data := req.Data()
	size := core.FrameHeaderLen + len(req.Data())
	m, ok := req.Metadata()
	if ok {
		size += 3 + len(m)
	}
	sid := dc.nextStreamID()

	releasable, isReleasable := req.(common.Releasable)
	if isReleasable {
		releasable.IncRef()
	}

	if !dc.shouldSplit(size) {
		outMsg := framing.NewWriteableFireAndForgetFrame(sid, data, m, 0)
		if isReleasable {
			outMsg.HandleDone(func() {
				releasable.Release()
			})
		}
		dc.sendFrame(outMsg)
		return
	}
	dc.doSplit(data, m, func(index int, result fragmentation.SplitResult) {
		var outMsg core.WriteableFrame
		if index == 0 {
			outMsg = framing.NewWriteableFireAndForgetFrame(sid, result.Data, result.Metadata, result.Flag)
		} else {
			outMsg = framing.NewWriteablePayloadFrame(sid, result.Data, result.Metadata, result.Flag|core.FlagNext)
		}

		if !result.Flag.Check(core.FlagFollow) && isReleasable {
			releasable.Release()
		}
		dc.sendFrame(outMsg)
	})
}

// MetadataPush start a request of MetadataPush.
func (dc *DuplexConnection) MetadataPush(payload payload.Payload) {
	if dc.closed.Load() {
		return
	}
	metadata, _ := payload.Metadata()
	dc.sendFrame(framing.NewWriteableMetadataPushFrame(metadata))
}

// RequestResponse start a request of RequestResponse.
func (dc *DuplexConnection) RequestResponse(req payload.Payload) (res mono.Mono) {
	if dc.closed.Load() {
		res = mono.Error(errSocketClosed)
		return
	}

	sid := dc.nextStreamID()

	handler := &requestResponseCallback{}

	onFinally := func(s reactor.SignalType, d reactor.Disposable) {
		common.TryRelease(handler.cache)
		if s == reactor.SignalTypeCancel {
			dc.sendFrame(framing.NewWriteableCancelFrame(sid))
		}
		// Unregister handler w/sink (processor).
		dc.unregister(sid)
		// Dispose sink (processor).
		d.Dispose()
	}

	m, s, _ := mono.NewProcessor(dc.reqSche, onFinally)
	handler.sink = s

	dc.register(sid, handler)

	res = m

	data := req.Data()
	metadata, _ := req.Metadata()

	// sending...
	size := framing.CalcPayloadFrameSize(data, metadata)

	releasable, isReleasable := req.(common.Releasable)
	if isReleasable {
		releasable.IncRef()
	}

	// mtu disabled
	if !dc.shouldSplit(size) {
		toBeSent := framing.NewWriteableRequestResponseFrame(sid, data, metadata, 0)
		if isReleasable {
			toBeSent.HandleDone(func() {
				releasable.Release()
			})
		}
		if ok := dc.sendFrame(toBeSent); !ok {
			dc.killCallback(sid)
		}
		return
	}

	// mtu enabled
	dc.doSplit(data, metadata, func(index int, result fragmentation.SplitResult) {
		var toBeSent core.WriteableFrame
		if index == 0 {
			toBeSent = framing.NewWriteableRequestResponseFrame(sid, result.Data, result.Metadata, result.Flag)
		} else {
			toBeSent = framing.NewWriteablePayloadFrame(sid, result.Data, result.Metadata, result.Flag|core.FlagNext)
		}

		// Add release hook at last frame.
		if !result.Flag.Check(core.FlagFollow) && isReleasable {
			toBeSent.HandleDone(func() {
				releasable.Release()
			})
		}

		if ok := dc.sendFrame(toBeSent); !ok {
			dc.killCallback(sid)
		}
	})

	return
}

// RequestStream start a request of RequestStream.
func (dc *DuplexConnection) RequestStream(sending payload.Payload) (ret flux.Flux) {
	if dc.closed.Load() {
		ret = flux.Error(errSocketClosed)
		return
	}

	sid := dc.nextStreamID()
	pc := flux.CreateProcessor()

	dc.register(sid, requestStreamCallback{pc: pc})

	requested := atomic.NewBool(false)

	// Create a queue to save those payloads to be released.
	toBeReleased := queue.NewLKQueue()

	ret = pc.
		DoFinally(func(sig rx.SignalType) {
			if sig == rx.SignalCancel {
				dc.sendFrame(framing.NewWriteableCancelFrame(sid))
			}
			dc.unregister(sid)
			for {
				next := toBeReleased.Dequeue()
				if next == nil {
					break
				}
				next.(common.Releasable).Release()
			}
		}).
		DoOnNext(func(input payload.Payload) error {
			if nextRelease := toBeReleased.Dequeue(); nextRelease != nil {
				nextRelease.(common.Releasable).Release()
			}
			if _, ok := input.(common.Releasable); ok {
				toBeReleased.Enqueue(input)
			}
			return nil
		}).
		DoOnRequest(func(n int) {
			n32 := ToUint32RequestN(n)

			// Send RequestN at first time.
			if !requested.CAS(false, true) {
				done := make(chan struct{})
				frameN := framing.NewWriteableRequestNFrame(sid, n32, 0)
				frameN.HandleDone(func() {
					close(done)
				})
				if dc.sendFrame(frameN) {
					<-done
				}
				return
			}

			releasable, isReleasable := sending.(common.Releasable)

			if isReleasable {
				releasable.IncRef()
			}

			data := sending.Data()
			metadata, _ := sending.Metadata()

			size := framing.CalcPayloadFrameSize(data, metadata) + 4
			if !dc.shouldSplit(size) {
				toBeSent := framing.NewWriteableRequestStreamFrame(sid, n32, data, metadata, 0)

				if isReleasable {
					toBeSent.HandleDone(func() {
						releasable.Release()
					})
				}

				if ok := dc.sendFrame(toBeSent); !ok {
					dc.killCallback(sid)
				}
				return
			}

			dc.doSplitSkip(4, data, metadata, func(index int, result fragmentation.SplitResult) {
				var toBeSent core.WriteableFrame
				if index == 0 {
					toBeSent = framing.NewWriteableRequestStreamFrame(sid, n32, result.Data, result.Metadata, result.Flag)
				} else {
					toBeSent = framing.NewWriteablePayloadFrame(sid, result.Data, result.Metadata, result.Flag|core.FlagNext)
				}

				// Add release hook at last frame.
				if !result.Flag.Check(core.FlagFollow) && isReleasable {
					toBeSent.HandleDone(func() {
						releasable.Release()
					})
				}

				if ok := dc.sendFrame(toBeSent); !ok {
					dc.killCallback(sid)
				}
			})
		})
	return
}

func (dc *DuplexConnection) killCallback(sid uint32) {
	cb, ok := dc.messages.Load(sid)
	if !ok {
		return
	}
	cb.(callback).stopWithError(errRequestFailed)
}

// RequestChannel start a request of RequestChannel.
func (dc *DuplexConnection) RequestChannel(request payload.Payload, sending flux.Flux) (ret flux.Flux) {
	if dc.closed.Load() {
		ret = flux.Error(errSocketClosed)
		return
	}

	sid := dc.nextStreamID()

	receiving := flux.CreateProcessor()

	rcvRequested := atomic.NewBool(false)

	toBeReleased := queue.NewLKQueue()

	ret = receiving.
		DoFinally(func(sig rx.SignalType) {
			dc.unregister(sid)
			// release resources.
			for {
				next := toBeReleased.Dequeue()
				if next == nil {
					break
				}
				next.(common.Releasable).Release()
			}
		}).
		DoOnNext(func(next payload.Payload) error {
			if nextRelease := toBeReleased.Dequeue(); nextRelease != nil {
				nextRelease.(common.Releasable).Release()
			}
			if _, ok := next.(common.Releasable); ok {
				toBeReleased.Enqueue(next)
			}
			return nil
		}).
		DoOnRequest(func(initN int) {
			n := ToUint32RequestN(initN)
			isFirstRequest := rcvRequested.CAS(false, true)
			if !isFirstRequest {
				// block send RequestN frame
				frameN := framing.NewWriteableRequestNFrame(sid, n, 0)
				done := make(chan struct{})
				frameN.HandleDone(func() {
					close(done)
				})
				if dc.sendFrame(frameN) {
					<-done
				}
				return
			}

			// First request - send the initial REQUEST_CHANNEL frame with the request payload,
			// then subscribe to the sending flux for subsequent payloads.
			releasable, isReleasable := request.(common.Releasable)

			if isReleasable {
				releasable.IncRef()
			}

			data := request.Data()
			metadata, _ := request.Metadata()

			size := framing.CalcPayloadFrameSize(data, metadata) + 4
			if !dc.shouldSplit(size) {
				toBeSent := framing.NewWriteableRequestChannelFrame(sid, n, data, metadata, core.FlagNext)

				if isReleasable {
					toBeSent.HandleDone(func() {
						releasable.Release()
					})
				}

				if ok := dc.sendFrame(toBeSent); !ok {
					dc.killCallback(sid)
					return
				}
			} else {
				dc.doSplitSkip(4, data, metadata, func(index int, result fragmentation.SplitResult) {
					var toBeSent core.WriteableFrame
					if index == 0 {
						toBeSent = framing.NewWriteableRequestChannelFrame(sid, n, result.Data, result.Metadata, result.Flag|core.FlagNext)
					} else {
						toBeSent = framing.NewWriteablePayloadFrame(sid, result.Data, result.Metadata, result.Flag|core.FlagNext)
					}

					// Add release hook at last frame.
					if !result.Flag.Check(core.FlagFollow) && isReleasable {
						toBeSent.HandleDone(func() {
							releasable.Release()
						})
					}

					if ok := dc.sendFrame(toBeSent); !ok {
						dc.killCallback(sid)
					}
				})
			}

			// Subscribe to sending flux for subsequent payloads
			sub := &requestChannelSubscriber{
				sid: sid,
				dc:  dc,
				rcv: receiving,
			}
			sending.SubscribeOn(dc.reqSche).SubscribeWith(dc.ctx, sub)
		})
	return ret
}

func (dc *DuplexConnection) onFrameRequestResponse(frame core.BufferedFrame) error {
	// fragment
	receiving, ok := dc.doFragment(frame.(*framing.RequestResponseFrame))
	if !ok {
		return nil
	}
	return dc.respondRequestResponse(receiving)
}

func (dc *DuplexConnection) respondRequestResponse(receiving fragmentation.HeaderAndPayload) error {
	sid := receiving.Header().StreamID()

	// execute socket handler
	sending, err := func() (resp mono.Mono, err error) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if e, ok := rec.(error); ok {
				err = errors.WithStack(e)
			} else {
				err = errors.Errorf("%v", rec)
			}
			logger.Errorf("handle request-response failed: %+v\n", err)
		}()
		resp = dc.responder.RequestResponse(receiving)
		if resp == nil {
			err = framing.NewWriteableErrorFrame(sid, core.ErrorCodeApplicationError, unsupportedRequestResponse)
		}
		return
	}()
	// sending error with panic
	if err != nil {
		common.TryRelease(receiving)
		dc.writeError(sid, err)
		return nil
	}

	// async subscribe publisher
	sub := borrowRequestResponseSubscriber(dc, sid, receiving)
	if mono.IsSubscribeAsync(sending) {
		sending.SubscribeWith(dc.ctx, sub)
		return nil
	}
	mustExecute(dc.resSche, func() {
		sending.SubscribeWith(dc.ctx, sub)
	})
	return nil
}

func (dc *DuplexConnection) onFrameRequestChannel(input core.BufferedFrame) error {
	receiving, ok := dc.doFragment(input.(*framing.RequestChannelFrame))
	if !ok {
		return nil
	}
	return dc.respondRequestChannel(receiving)
}

func (dc *DuplexConnection) respondRequestChannel(req fragmentation.HeaderAndPayload) error {
	// seek initRequestN
	initRequestN := extractRequestStreamInitN(req)

	sid := req.Header().StreamID()
	receivingProcessor := flux.CreateProcessor()

	finallyRequests := atomic.NewInt32(0)

	toBeReleased := queue.NewLKQueue()

	receiving := receivingProcessor.
		DoFinally(func(sig rx.SignalType) {
			if finallyRequests.Inc() == 2 {
				dc.unregister(sid)
			}
			if sig == rx.SignalCancel {
				dc.sendFrame(framing.NewWriteableCancelFrame(sid))
			}
			for {
				next := toBeReleased.Dequeue()
				if next == nil {
					break
				}
				next.(common.Releasable).Release()
			}
		}).
		DoOnNext(func(input payload.Payload) error {
			if nextRelease := toBeReleased.Dequeue(); nextRelease != nil {
				nextRelease.(common.Releasable).Release()
			}
			if _, ok := input.(common.Releasable); ok {
				toBeReleased.Enqueue(input)
			}
			return nil
		}).
		DoOnRequest(func(n int) {
			frameN := framing.NewWriteableRequestNFrame(sid, ToUint32RequestN(n), 0)
			done := make(chan struct{})
			frameN.HandleDone(func() {
				close(done)
			})
			if dc.sendFrame(frameN) {
				<-done
			}
		}).
		SubscribeOn(dc.reqSche)

	// TODO: if receiving == sending ???
	sending, err := func() (resp flux.Flux, err error) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if e, ok := rec.(error); ok {
				err = errors.WithStack(e)
			} else {
				err = errors.Errorf("%v", rec)
			}
			logger.Errorf("handle request-channel failed: %+v\n", err)
		}()
		resp = dc.responder.RequestChannel(req, receiving)
		if resp == nil {
			err = framing.NewWriteableErrorFrame(sid, core.ErrorCodeApplicationError, unsupportedRequestChannel)
		}
		return
	}()

	if err != nil {
		common.TryRelease(receiving)
		dc.writeError(sid, err)
		return nil
	}

	// Ensure registering message success before func end.
	subscribed := make(chan struct{})

	// Create subscriber
	sub := &respondChannelSubscriber{
		sid:        sid,
		n:          initRequestN,
		dc:         dc,
		rcv:        receivingProcessor,
		subscribed: subscribed,
		calls:      finallyRequests,
	}

	mustExecute(dc.reqSche, func() {
		sending.SubscribeWith(dc.ctx, sub)
	})

	<-subscribed

	return nil
}

func (dc *DuplexConnection) respondMetadataPush(input core.BufferedFrame) error {
	req := input.(*framing.MetadataPushFrame)
	mustExecute(dc.resSche, func() {
		defer func() {
			req.Release()
			rec := recover()
			if rec == nil {
				return
			}
			var err error
			if e, ok := rec.(error); ok {
				err = errors.WithStack(e)
			} else {
				err = errors.Errorf("%v", rec)
			}
			logger.Errorf("handle metadata-push failed: %+v\n", err)
		}()
		dc.responder.MetadataPush(req)
	})
	return nil
}

func (dc *DuplexConnection) onFrameFNF(frame core.BufferedFrame) error {
	receiving, ok := dc.doFragment(frame.(*framing.FireAndForgetFrame))
	if !ok {
		return nil
	}
	return dc.respondFireAndForget(receiving)
}

func (dc *DuplexConnection) respondFireAndForget(receiving fragmentation.HeaderAndPayload) error {
	mustExecute(dc.resSche, func() {
		defer func() {
			common.TryRelease(receiving)
			rec := recover()
			if rec == nil {
				return
			}
			var err error
			if e, ok := rec.(error); ok {
				err = errors.WithStack(e)
			} else {
				err = errors.Errorf("%v", rec)
			}
			logger.Errorf("handle fire-and-forget failed: %+v\n", err)
		}()
		dc.responder.FireAndForget(receiving)
	})
	return nil
}

func (dc *DuplexConnection) onFrameRequestStream(frame core.BufferedFrame) error {
	receiving, ok := dc.doFragment(frame.(*framing.RequestStreamFrame))
	if !ok {
		return nil

	}
	return dc.respondRequestStream(receiving)
}

func (dc *DuplexConnection) respondRequestStream(receiving fragmentation.HeaderAndPayload) error {
	sid := receiving.Header().StreamID()
	n := extractRequestStreamInitN(receiving)

	// execute request stream handler
	sending, err := func() (resp flux.Flux, err error) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			if e, ok := rec.(error); ok {
				err = errors.WithStack(e)
			} else {
				err = errors.Errorf("%v", rec)
			}
			logger.Errorf("handle request-stream failed: %+v\n", err)
		}()
		resp = dc.responder.RequestStream(receiving)
		if resp == nil {
			err = framing.NewWriteableErrorFrame(sid, core.ErrorCodeApplicationError, unsupportedRequestStream)
		}
		return
	}()

	// send error with panic
	if err != nil {
		common.TryRelease(receiving)
		dc.writeError(sid, err)
		return nil
	}

	// async subscribe publisher
	sub := borrowRequestStreamSubscriber(receiving, dc, sid, n)
	sending.SubscribeOn(dc.resSche).SubscribeWith(dc.ctx, sub)
	return nil
}

func (dc *DuplexConnection) writeError(sid uint32, e error) {
	// ignore sending error because current socket has been closed.
	if IsSocketClosedError(e) {
		return
	}
	switch err := e.(type) {
	case *framing.WriteableErrorFrame:
		dc.sendFrame(err)
	case core.CustomError:
		dc.sendFrame(framing.NewWriteableErrorFrame(sid, err.ErrorCode(), err.ErrorData()))
	default:
		errFrame := framing.NewWriteableErrorFrame(
			sid,
			core.ErrorCodeApplicationError,
			bytesconv.StringToBytes(e.Error()),
		)
		dc.sendFrame(errFrame)
	}
}

// SetResponder sets a responder for current socket.
func (dc *DuplexConnection) SetResponder(responder Responder) {
	dc.responder = responder
}

func (dc *DuplexConnection) onFrameKeepalive(frame core.BufferedFrame) (err error) {
	defer frame.Release()
	f := frame.(*framing.KeepaliveFrame)
	if !f.HasFlag(core.FlagRespond) {
		return
	}
	// TODO: optimize, if keepalive frame support modify data.
	data := common.CloneBytes(f.Data())
	k := framing.NewWriteableKeepaliveFrame(f.LastReceivedPosition(), data, false)
	dc.sendFrame(k)
	return
}

func (dc *DuplexConnection) deleteFragment(sid uint32) {
	v, ok := dc.fragments.Load(sid)
	if !ok {
		return
	}
	dc.fragments.Delete(sid)
	common.TryRelease(v)
}

func (dc *DuplexConnection) onFrameCancel(frame core.BufferedFrame) (err error) {
	sid := frame.Header().StreamID()
	frame.Release()

	defer dc.deleteFragment(sid)

	v, ok := dc.messages.Load(sid)
	if !ok {
		logger.Warnf("unmatched frame CANCEL(id=%d), maybe original request has been cancelled\n", sid)
		return
	}

	switch vv := v.(type) {
	case requestResponseCallbackReverse:
		vv.su.Cancel()
	case requestStreamCallbackReverse:
		vv.su.Cancel()
	case respondChannelCallback:
		vv.snd.Cancel()
	default:
		panic(fmt.Sprintf("unreachable: should never occur %T!", vv))
	}

	return
}

func (dc *DuplexConnection) onFrameError(input core.BufferedFrame) error {
	defer input.Release()
	f := input.(*framing.ErrorFrame)
	sid := f.Header().StreamID()

	// TODO: avoid clone error
	err := f.ToError()

	v, ok := dc.messages.Load(sid)
	if !ok {
		dc.deleteFragment(sid)
		logger.Warnf("unmatched frame ERROR(id=%d), maybe original request has been cancelled\n", sid)
		return nil
	}

	switch vv := v.(type) {
	case *requestResponseCallback:
		vv.sink.Error(err)
	case requestStreamCallback:
		vv.pc.Error(err)
	case requestChannelCallback:
		vv.rcv.Error(err)
	default:
		return errors.Errorf("illegal value for error: %v", vv)
	}
	return nil
}

func (dc *DuplexConnection) onFrameRequestN(input core.BufferedFrame) error {
	defer input.Release()
	f := input.(*framing.RequestNFrame)
	sid := f.Header().StreamID()
	v, ok := dc.messages.Load(sid)
	if !ok {
		dc.deleteFragment(sid)
		logger.Warnf("unmatched frame REQUEST_N(id=%d), maybe original request has been cancelled\n", sid)
		return nil
	}
	n := ToIntRequestN(f.N())
	switch vv := v.(type) {
	case requestStreamCallbackReverse:
		vv.su.Request(n)
	case requestChannelCallback:
		vv.snd.Request(n)
	case respondChannelCallback:
		vv.snd.Request(n)
	}
	return nil
}

func (dc *DuplexConnection) doFragment(input fragmentation.HeaderAndPayload) (out fragmentation.HeaderAndPayload, ok bool) {
	h := input.Header()
	sid := h.StreamID()
	v, exist := dc.fragments.Load(sid)
	if exist {
		joiner := v.(fragmentation.Joiner)
		ok = joiner.Push(input)
		if ok {
			dc.fragments.Delete(sid)
			out = joiner
		}
		return
	}
	ok = !h.Flag().Check(core.FlagFollow)
	if ok {
		out = input
		return
	}
	dc.fragments.Store(sid, fragmentation.NewJoiner(input))
	return
}

func (dc *DuplexConnection) onFramePayload(frame core.BufferedFrame) error {
	next, ok := dc.doFragment(frame.(*framing.PayloadFrame))
	if !ok {
		return nil
	}
	h := next.Header()

	switch h.Type() {
	case core.FrameTypeRequestFNF:
		return dc.respondFireAndForget(next)
	case core.FrameTypeRequestResponse:
		return dc.respondRequestResponse(next)
	case core.FrameTypeRequestStream:
		return dc.respondRequestStream(next)
	case core.FrameTypeRequestChannel:
		return dc.respondRequestChannel(next)
	}

	sid := h.StreamID()
	v, ok := dc.messages.Load(sid)
	if !ok {
		common.TryRelease(next)
		logger.Warnf("unmatched frame PAYLOAD(id=%d), maybe original request has been cancelled\n", sid)
		return nil
	}

	switch handler := v.(type) {
	case *requestResponseCallback:
		handler.cache = next
		handler.sink.Success(next)
	case requestStreamCallback:
		fg := h.Flag()
		isNext := fg.Check(core.FlagNext)
		if isNext {
			handler.pc.Next(next)
		}
		if fg.Check(core.FlagComplete) {
			if !isNext {
				common.TryRelease(next)
			}
			// Release pure complete payload
			handler.pc.Complete()
		}
	case requestChannelCallback:
		fg := h.Flag()
		isNext := fg.Check(core.FlagNext)
		if isNext {
			handler.rcv.Next(next)
		}
		if fg.Check(core.FlagComplete) {
			if !isNext {
				common.TryRelease(next)
			}
			handler.rcv.Complete()
		}
	case respondChannelCallback:
		fg := h.Flag()
		isNext := fg.Check(core.FlagNext)
		if isNext {
			handler.rcv.Next(next)
		}
		if fg.Check(core.FlagComplete) {
			if !isNext {
				common.TryRelease(next)
			}
			handler.rcv.Complete()
		}
	}
	return nil
}

func (dc *DuplexConnection) clearTransport() {
	dc.locker.Lock()
	defer dc.locker.Unlock()
	dc.tp = nil
	dc.ready.Store(false)
}

func (dc *DuplexConnection) currentTransport() (tp *transport.Transport) {
	dc.locker.RLock()
	tp = dc.tp
	dc.locker.RUnlock()
	return
}

// SetTransport sets a transport for current socket.
func (dc *DuplexConnection) SetTransport(tp *transport.Transport) (ok bool) {
	tp.Handle(transport.OnCancel, dc.onFrameCancel)
	tp.Handle(transport.OnError, dc.onFrameError)
	tp.Handle(transport.OnRequestN, dc.onFrameRequestN)
	tp.Handle(transport.OnPayload, dc.onFramePayload)
	tp.Handle(transport.OnKeepalive, dc.onFrameKeepalive)
	if dc.responder != nil {
		tp.Handle(transport.OnRequestResponse, dc.onFrameRequestResponse)
		tp.Handle(transport.OnMetadataPush, dc.respondMetadataPush)
		tp.Handle(transport.OnFireAndForget, dc.onFrameFNF)
		tp.Handle(transport.OnRequestStream, dc.onFrameRequestStream)
		tp.Handle(transport.OnRequestChannel, dc.onFrameRequestChannel)
	}

	ok = dc.ready.CAS(false, true)
	if !ok {
		return
	}

	dc.locker.Lock()
	dc.tp = tp
	dc.cond.Signal()
	dc.locker.Unlock()
	return
}

func (dc *DuplexConnection) sendFrame(f core.WriteableFrame) (ok bool) {
	if dc.closed.Load() {
		f.Done()
		return
	}
	defer func() {
		if e := recover(); e == nil {
			ok = true
		} else {
			f.Done()
		}
	}()
	dc.sndQueue <- f
	return
}

func (dc *DuplexConnection) sendPayload(
	sid uint32,
	sending payload.Payload,
	frameFlag core.FrameFlag,
) {
	d := sending.Data()
	m, _ := sending.Metadata()
	size := framing.CalcPayloadFrameSize(d, m)

	releasable, isReleasable := sending.(common.Releasable)
	if isReleasable {
		releasable.IncRef()
	}

	if !dc.shouldSplit(size) {
		toBeSent := framing.NewWriteablePayloadFrame(sid, d, m, frameFlag)
		if isReleasable {
			toBeSent.HandleDone(func() {
				releasable.Release()
			})
		}
		dc.sendFrame(toBeSent)
		return
	}
	dc.doSplit(d, m, func(index int, result fragmentation.SplitResult) {
		flag := result.Flag
		if index == 0 {
			flag |= frameFlag
		} else {
			flag |= core.FlagNext
		}

		// lazy release at last frame
		next := framing.NewWriteablePayloadFrame(sid, result.Data, result.Metadata, flag)

		if !result.Flag.Check(core.FlagFollow) && isReleasable {
			next.HandleDone(func() {
				releasable.Release()
			})
		}
		dc.sendFrame(next)
	})
}

func (dc *DuplexConnection) drainWithKeepaliveAndLease(leaseChan <-chan lease.Lease) (ok bool) {
	if len(dc.sndQueue) > 0 {
		dc.drain(nil)
	}
	var out core.WriteableFrame
	select {
	case <-dc.keepaliver.C():
		ok = true
		out = framing.NewWriteableKeepaliveFrame(dc.counter.ReadBytes(), nil, true)
		if tp := dc.currentTransport(); tp != nil {
			err := tp.Send(out, true)
			if err != nil {
				logger.Errorf("send keepalive frame failed: %s\n", err.Error())
			}
		}
	case ls, success := <-leaseChan:
		ok = success
		if !ok {
			return
		}
		out = framing.NewWriteableLeaseFrame(ls.TimeToLive, ls.NumberOfRequests, ls.Metadata)
		if tp := dc.currentTransport(); tp == nil {
			dc.sndBacklog = append(dc.sndBacklog, out)
		} else if err := tp.Send(out, true); err != nil {
			logger.Errorf("send frame failed: %s\n", err.Error())
			dc.sndBacklog = append(dc.sndBacklog, out)
		}
	case out, ok = <-dc.sndQueue:
		if !ok {
			return
		}
		if tp := dc.currentTransport(); tp == nil {
			dc.sndBacklog = append(dc.sndBacklog, out)
		} else if err := tp.Send(out, true); err != nil {
			logger.Errorf("send frame failed: %s\n", err.Error())
			dc.sndBacklog = append(dc.sndBacklog, out)
		}
	}
	return
}

func (dc *DuplexConnection) drainWithKeepalive() (ok bool) {
	if len(dc.sndQueue) > 0 {
		dc.drain(nil)
	}
	var out core.WriteableFrame

	select {
	case <-dc.keepaliver.C():
		ok = true
		out = framing.NewWriteableKeepaliveFrame(dc.counter.ReadBytes(), nil, true)
		tp := dc.tp
		if tp == nil {
			return
		}
		err := tp.Send(out, true)
		if err != nil {
			logger.Errorf("send keepalive frame failed: %s\n", err.Error())
		}
	case out, ok = <-dc.sndQueue:
		if !ok {
			return
		}

		if tp := dc.currentTransport(); tp == nil {
			dc.sndBacklog = append(dc.sndBacklog, out)
		} else if err := tp.Send(out, true); err != nil {
			logger.Errorf("send frame failed: %s\n", err.Error())
			dc.sndBacklog = append(dc.sndBacklog, out)
		}
	}
	return
}

func (dc *DuplexConnection) drain(leaseChan <-chan lease.Lease) bool {
	var flush bool
	cycle := len(dc.sndQueue)
	if cycle < 1 {
		runtime.Gosched()
		cycle = 1
	}
	for i := 0; i < cycle; i++ {
		select {
		case next, ok := <-leaseChan:
			if !ok {
				return false
			}
			if dc.drainOne(framing.NewWriteableLeaseFrame(next.TimeToLive, next.NumberOfRequests, next.Metadata)) {
				flush = true
			}
		case out, ok := <-dc.sndQueue:
			if !ok {
				return false
			}
			if dc.drainOne(out) {
				flush = true
			}
		}
	}
	if flush {
		if err := dc.tp.Flush(); err != nil {
			logger.Errorf("flush failed: %v\n", err)
		}
	}
	return true
}

func (dc *DuplexConnection) drainOne(out core.WriteableFrame) (ok bool) {
	tp := dc.currentTransport()
	if tp == nil {
		dc.sndBacklog = append(dc.sndBacklog, out)
		return
	}
	err := tp.Send(out, false)
	if err != nil {
		dc.sndBacklog = append(dc.sndBacklog, out)
		logger.Errorf("send frame failed: %s\n", err.Error())
		return
	}
	ok = true
	return
}

func (dc *DuplexConnection) drainBacklog() {
	if len(dc.sndBacklog) < 1 {
		return
	}
	defer func() {
		dc.sndBacklog = dc.sndBacklog[:0]
	}()

	dc.locker.RLock()
	tp := dc.tp
	dc.locker.RUnlock()
	if tp == nil {
		return
	}
	var out core.WriteableFrame
	for i := range dc.sndBacklog {
		out = dc.sndBacklog[i]
		if err := tp.Send(out, false); err != nil {
			out.Done()
			logger.Errorf("send frame failed: %v\n", err)
		}
	}
	if err := tp.Flush(); err != nil {
		logger.Errorf("flush failed: %v\n", err)
	}
}

func (dc *DuplexConnection) loopWriteWithKeepaliver(ctx context.Context, leaseChan <-chan lease.Lease) error {
	for {
		if dc.closed.Load() {
			break
		}
		if !dc.ready.Load() {
			dc.locker.Lock()
			dc.cond.Wait()
			dc.locker.Unlock()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// ignore
		}

		select {
		case <-dc.keepaliver.C():
			kf := framing.NewWriteableKeepaliveFrame(dc.counter.ReadBytes(), nil, true)
			if tp := dc.currentTransport(); tp != nil {
				err := tp.Send(kf, true)
				if err != nil {
					logger.Errorf("send keepalive frame failed: %s\n", err.Error())
				}
			}
		default:
		}

		dc.drainBacklog()
		if leaseChan == nil && !dc.drainWithKeepalive() {
			break
		}
		if leaseChan != nil && !dc.drainWithKeepaliveAndLease(leaseChan) {
			break
		}
	}
	return nil
}

// LoopWrite start write loop
func (dc *DuplexConnection) LoopWrite(ctx context.Context) error {
	// init write done chan
	dc.locker.Lock()
	if dc.writeDone == nil {
		dc.writeDone = make(chan struct{})
	}
	dc.locker.Unlock()

	defer close(dc.writeDone)

	var leaseChan chan lease.Lease
	if dc.leases != nil {
		leaseCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		if c, ok := dc.leases.Next(leaseCtx); ok {
			leaseChan = c
		}
	}

	if dc.keepaliver != nil {
		defer dc.keepaliver.Stop()
		return dc.loopWriteWithKeepaliver(ctx, leaseChan)
	}
	for {
		if dc.closed.Load() {
			break
		}
		if !dc.ready.Load() {
			dc.cond.L.Lock()
			dc.cond.Wait()
			dc.cond.L.Unlock()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		dc.drainBacklog()
		if !dc.drain(leaseChan) {
			break
		}
	}
	return nil
}

func (dc *DuplexConnection) doSplit(data, metadata []byte, handler fragmentation.HandleSplitResult) {
	fragmentation.Split(dc.mtu, data, metadata, handler)
}

func (dc *DuplexConnection) doSplitSkip(skip int, data, metadata []byte, handler fragmentation.HandleSplitResult) {
	fragmentation.SplitSkip(dc.mtu, skip, data, metadata, handler)
}

func (dc *DuplexConnection) shouldSplit(size int) bool {
	return size > dc.mtu
}

func (dc *DuplexConnection) register(sid uint32, msg interface{}) {
	dc.messages.Store(sid, msg)
}

func (dc *DuplexConnection) unregister(sid uint32) {
	dc.messages.Delete(sid)
	dc.deleteFragment(sid)
}

// IsSocketClosedError returns true if input error is for socket closed.
func IsSocketClosedError(err error) bool {
	return err == errSocketClosed
}

// NewServerDuplexConnection creates a new server-side DuplexConnection.
func NewServerDuplexConnection(ctx context.Context, reqSche, resSche scheduler.Scheduler, mtu int, leases lease.Factory) *DuplexConnection {
	return newDuplexConnection(ctx, reqSche, resSche, mtu, nil, &serverStreamIDs{}, leases)
}

// NewClientDuplexConnection creates a new client-side DuplexConnection.
func NewClientDuplexConnection(ctx context.Context, reqSche, resSche scheduler.Scheduler, mtu int, keepaliveInterval time.Duration) *DuplexConnection {
	return newDuplexConnection(ctx, reqSche, resSche, mtu, NewKeepaliver(keepaliveInterval), &clientStreamIDs{}, nil)
}

func newDuplexConnection(ctx context.Context, reqSche, resSche scheduler.Scheduler, mtu int, ka *Keepaliver, sids StreamID, leases lease.Factory) *DuplexConnection {
	destroyReqSche := reqSche == nil
	if destroyReqSche {
		reqSche = scheduler.NewElastic(misc.MaxInt(runtime.NumCPU()<<8, _minRequestSchedulerSize))
	}
	if resSche == nil {
		resSche = scheduler.Elastic()
	}

	c := &DuplexConnection{
		ctx:            ctx,
		reqSche:        reqSche,
		resSche:        resSche,
		destroyReqSche: destroyReqSche,
		leases:         leases,
		sndQueue:       make(chan core.WriteableFrame, _outChanSize),
		mtu:            mtu,
		sids:           sids,
		counter:        core.NewTrafficCounter(),
		keepaliver:     ka,
		closed:         atomic.NewBool(false),
		ready:          atomic.NewBool(false),
	}

	c.cond.L = &c.locker
	return c
}
