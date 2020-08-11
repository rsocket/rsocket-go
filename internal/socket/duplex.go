package socket

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/internal/fragmentation"
	"github.com/rsocket/rsocket-go/lease"
	"github.com/rsocket/rsocket-go/logger"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/rsocket/rsocket-go/rx/mono"
)

const _outChanSize = 64

var errSocketClosed = errors.New("socket closed already")

var (
	unsupportedRequestStream   = []byte("Request-Stream not implemented.")
	unsupportedRequestResponse = []byte("Request-Response not implemented.")
	unsupportedRequestChannel  = []byte("Request-Channel not implemented.")
)

// IsSocketClosedError returns true if input error is for socket closed.
func IsSocketClosedError(err error) bool {
	return err == errSocketClosed
}

// DuplexConnection represents a socket of RSocket which can be a requester or a responder.
type DuplexConnection struct {
	l               *sync.RWMutex
	counter         *core.TrafficCounter
	tp              *transport.Transport
	outs            chan core.WriteableFrame
	outsPriority    []core.WriteableFrame
	responder       Responder
	messages        *sync.Map
	sids            StreamID
	mtu             int
	fragments       *sync.Map // common.U32Map // key=streamID, value=Joiner
	writeDone       chan struct{}
	keepaliver      *Keepaliver
	cond            *sync.Cond
	singleScheduler scheduler.Scheduler
	e               error
	leases          lease.Leases
	closeOnce       sync.Once
}

// SetError sets error for current socket.
func (dc *DuplexConnection) SetError(err error) {
	dc.l.Lock()
	defer dc.l.Unlock()
	dc.e = err
}

// GetError get the error set.
func (dc *DuplexConnection) GetError() error {
	dc.l.RLock()
	defer dc.l.RUnlock()
	return dc.e
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
func (dc *DuplexConnection) Close() (err error) {
	dc.closeOnce.Do(func() {
		err = dc.innerClose()
	})
	return
}

func (dc *DuplexConnection) innerClose() error {
	if dc.keepaliver != nil {
		dc.keepaliver.Stop()
	}
	_ = dc.singleScheduler.Close()
	close(dc.outs)
	dc.cond.L.Lock()
	dc.cond.Broadcast()
	dc.cond.L.Unlock()

	<-dc.writeDone

	if dc.tp != nil {
		if dc.e == nil {
			dc.e = dc.tp.Close()
		} else {
			_ = dc.tp.Close()
		}
	}
	dc.messages.Range(func(_, v interface{}) bool {
		if cb, ok := v.(callback); ok {
			err := dc.e
			if err == nil {
				err = errSocketClosed
			}
			go cb.Close(err)
		}
		return true
	})
	return dc.e
}

// FireAndForget start a request of FireAndForget.
func (dc *DuplexConnection) FireAndForget(sending payload.Payload) {
	data := sending.Data()
	size := core.FrameHeaderLen + len(sending.Data())
	m, ok := sending.Metadata()
	if ok {
		size += 3 + len(m)
	}
	sid := dc.nextStreamID()
	if !dc.shouldSplit(size) {
		dc.sendFrame(framing.NewWriteableFireAndForgetFrame(sid, data, m, 0))
		return
	}
	dc.doSplit(data, m, func(index int, result fragmentation.SplitResult) {
		var f core.WriteableFrame
		if index == 0 {
			f = framing.NewWriteableFireAndForgetFrame(sid, result.Data, result.Metadata, result.Flag)
		} else {
			f = framing.NewWriteablePayloadFrame(sid, result.Data, result.Metadata, result.Flag|core.FlagNext)
		}
		dc.sendFrame(f)
	})
}

// MetadataPush start a request of MetadataPush.
func (dc *DuplexConnection) MetadataPush(payload payload.Payload) {
	metadata, _ := payload.Metadata()
	dc.sendFrame(framing.NewWriteableMetadataPushFrame(metadata))
}

// RequestResponse start a request of RequestResponse.
func (dc *DuplexConnection) RequestResponse(pl payload.Payload) (mo mono.Mono) {
	sid := dc.nextStreamID()
	resp := mono.CreateProcessor()

	dc.register(sid, requestResponseCallback{pc: resp})

	data := pl.Data()
	metadata, _ := pl.Metadata()

	mo = resp.
		DoFinally(func(s rx.SignalType) {
			if s == rx.SignalCancel {
				dc.sendFrame(framing.NewWriteableCancelFrame(sid))
			}
			dc.unregister(sid)
		})

	// sending...
	size := framing.CalcPayloadFrameSize(data, metadata)
	if !dc.shouldSplit(size) {
		dc.sendFrame(framing.NewWriteableRequestResponseFrame(sid, data, metadata, 0))
		return
	}
	dc.doSplit(data, metadata, func(index int, result fragmentation.SplitResult) {
		var f core.WriteableFrame
		if index == 0 {
			f = framing.NewWriteableRequestResponseFrame(sid, result.Data, result.Metadata, result.Flag)
		} else {
			f = framing.NewWriteablePayloadFrame(sid, result.Data, result.Metadata, result.Flag|core.FlagNext)
		}
		dc.sendFrame(f)
	})

	return
}

// RequestStream start a request of RequestStream.
func (dc *DuplexConnection) RequestStream(sending payload.Payload) (ret flux.Flux) {
	sid := dc.nextStreamID()
	pc := flux.CreateProcessor()

	dc.register(sid, requestStreamCallback{pc: pc})

	requested := make(chan struct{})

	ret = pc.
		DoFinally(func(sig rx.SignalType) {
			if sig == rx.SignalCancel {
				dc.sendFrame(framing.NewWriteableCancelFrame(sid))
			}
			dc.unregister(sid)
		}).
		DoOnRequest(func(n int) {
			n32 := ToUint32RequestN(n)

			var newborn bool
			select {
			case <-requested:
			default:
				newborn = true
				close(requested)
			}

			if !newborn {
				frameN := framing.NewWriteableRequestNFrame(sid, n32, 0)
				dc.sendFrame(frameN)
				<-frameN.DoneNotify()
				return
			}

			data := sending.Data()
			metadata, _ := sending.Metadata()

			size := framing.CalcPayloadFrameSize(data, metadata) + 4
			if !dc.shouldSplit(size) {
				dc.sendFrame(framing.NewWriteableRequestStreamFrame(sid, n32, data, metadata, 0))
				return
			}

			dc.doSplitSkip(4, data, metadata, func(index int, result fragmentation.SplitResult) {
				var f core.WriteableFrame
				if index == 0 {
					f = framing.NewWriteableRequestStreamFrame(sid, n32, result.Data, result.Metadata, result.Flag)
				} else {
					f = framing.NewWriteablePayloadFrame(sid, result.Data, result.Metadata, result.Flag|core.FlagNext)
				}
				dc.sendFrame(f)
			})
		})
	return
}

// RequestChannel start a request of RequestChannel.
func (dc *DuplexConnection) RequestChannel(publisher rx.Publisher) (ret flux.Flux) {
	sid := dc.nextStreamID()

	sending := publisher.(flux.Flux)
	receiving := flux.CreateProcessor()

	rcvRequested := make(chan struct{})

	ret = receiving.
		DoFinally(func(sig rx.SignalType) {
			dc.unregister(sid)
		}).
		DoOnRequest(func(n int) {
			n32 := ToUint32RequestN(n)
			var newborn bool
			select {
			case <-rcvRequested:
			default:
				newborn = true
				close(rcvRequested)
			}
			if !newborn {
				frameN := framing.NewWriteableRequestNFrame(sid, n32, 0)
				dc.sendFrame(frameN)
				<-frameN.DoneNotify()
				return
			}

			sndRequested := make(chan struct{})
			sub := rx.NewSubscriber(
				rx.OnNext(func(item payload.Payload) (err error) {
					var newborn bool
					select {
					case <-sndRequested:
					default:
						newborn = true
						close(sndRequested)
					}
					if !newborn {
						dc.sendPayload(sid, item, core.FlagNext)
						return
					}

					d := item.Data()
					m, _ := item.Metadata()
					size := framing.CalcPayloadFrameSize(d, m) + 4
					if !dc.shouldSplit(size) {
						metadata, _ := item.Metadata()
						dc.sendFrame(framing.NewWriteableRequestChannelFrame(sid, n32, item.Data(), metadata, core.FlagNext))
						return
					}

					dc.doSplitSkip(4, d, m, func(index int, result fragmentation.SplitResult) {
						var f core.WriteableFrame
						if index == 0 {
							f = framing.NewWriteableRequestChannelFrame(sid, n32, result.Data, result.Metadata, result.Flag|core.FlagNext)
						} else {
							f = framing.NewWriteablePayloadFrame(sid, result.Data, result.Metadata, result.Flag|core.FlagNext)
						}
						dc.sendFrame(f)
					})
					return
				}),
				rx.OnSubscribe(func(s rx.Subscription) {
					dc.register(sid, requestChannelCallback{rcv: receiving, snd: s})
					s.Request(1)
				}),
			)
			sending.
				DoFinally(func(sig rx.SignalType) {
					// TODO: handle cancel or error
					switch sig {
					case rx.SignalComplete:
						complete := framing.NewPayloadFrame(sid, nil, nil, core.FlagComplete)
						dc.sendFrame(complete)
						<-complete.DoneNotify()
					default:
						panic(fmt.Errorf("unsupported sending channel signal: %s", sig))
					}
				}).
				SubscribeOn(scheduler.Parallel()).
				SubscribeWith(context.Background(), sub)
		})
	return ret
}

func (dc *DuplexConnection) onFrameRequestResponse(frame core.Frame) error {
	// fragment
	receiving, ok := dc.doFragment(frame.(*framing.RequestResponseFrame))
	if !ok {
		return nil
	}
	return dc.respondRequestResponse(receiving)
}

func (dc *DuplexConnection) respondRequestResponse(receiving fragmentation.HeaderAndPayload) error {
	sid := receiving.Header().StreamID()

	// 1. execute socket handler
	sending, err := func() (mono mono.Mono, err error) {
		defer func() {
			err = tryRecover(recover())
		}()
		mono = dc.responder.RequestResponse(receiving)
		return
	}()
	// 2. sending error with panic
	if err != nil {
		dc.writeError(sid, err)
		return nil
	}
	// 3. sending error with unsupported handler
	if sending == nil {
		dc.writeError(sid, framing.NewWriteableErrorFrame(sid, core.ErrorCodeApplicationError, unsupportedRequestResponse))
		return nil
	}

	// 4. async subscribe publisher
	sub := rx.NewSubscriber(
		rx.OnNext(func(input payload.Payload) error {
			dc.sendPayload(sid, input, core.FlagNext|core.FlagComplete)
			return nil
		}),
		rx.OnError(func(e error) {
			dc.writeError(sid, e)
		}),
		rx.OnSubscribe(func(s rx.Subscription) {
			dc.register(sid, requestResponseCallbackReverse{su: s})
			s.Request(rx.RequestMax)
		}),
	)
	sending.
		DoFinally(func(sig rx.SignalType) {
			dc.unregister(sid)
		}).
		SubscribeOn(scheduler.Parallel()).
		SubscribeWith(context.Background(), sub)
	return nil
}

func (dc *DuplexConnection) onFrameRequestChannel(input core.Frame) error {
	receiving, ok := dc.doFragment(input.(*framing.RequestChannelFrame))
	if !ok {
		return nil
	}
	return dc.respondRequestChannel(receiving)
}

func (dc *DuplexConnection) respondRequestChannel(pl fragmentation.HeaderAndPayload) error {
	// seek initRequestN
	var initRequestN int
	switch v := pl.(type) {
	case *framing.RequestChannelFrame:
		initRequestN = ToIntRequestN(v.InitialRequestN())
	case fragmentation.Joiner:
		initRequestN = ToIntRequestN(v.First().(*framing.RequestChannelFrame).InitialRequestN())
	default:
		panic("unreachable")
	}

	sid := pl.Header().StreamID()
	receivingProcessor := flux.CreateProcessor()

	ch := make(chan struct{}, 2)

	receiving := receivingProcessor.
		DoFinally(func(s rx.SignalType) {
			ch <- struct{}{}
			<-ch
			select {
			case _, ok := <-ch:
				if ok {
					close(ch)
					dc.unregister(sid)
				}
			default:
			}
		}).
		DoOnRequest(func(n int) {
			frameN := framing.NewWriteableRequestNFrame(sid, ToUint32RequestN(n), 0)
			dc.sendFrame(frameN)
			<-frameN.DoneNotify()
		})

	_ = dc.singleScheduler.Worker().Do(func() {
		receivingProcessor.Next(pl)
	})

	// TODO: if receiving == sending ???
	sending, err := func() (flux flux.Flux, err error) {
		defer func() {
			err = tryRecover(recover())
		}()
		flux = dc.responder.RequestChannel(receiving)
		if flux == nil {
			err = framing.NewWriteableErrorFrame(sid, core.ErrorCodeApplicationError, unsupportedRequestChannel)
		}
		return
	}()

	if err != nil {
		dc.writeError(sid, err)
		return nil
	}

	// Ensure registering message success before func end.
	mustSub := make(chan struct{})

	sub := rx.NewSubscriber(
		rx.OnError(func(e error) {
			dc.writeError(sid, e)
		}),
		rx.OnComplete(func() {
			complete := framing.NewPayloadFrame(sid, nil, nil, core.FlagComplete)
			dc.sendFrame(complete)
			<-complete.DoneNotify()
		}),
		rx.OnSubscribe(func(s rx.Subscription) {
			dc.register(sid, requestChannelCallbackReverse{rcv: receivingProcessor, snd: s})
			close(mustSub)
			s.Request(initRequestN)
		}),
		rx.OnNext(func(elem payload.Payload) error {
			dc.sendPayload(sid, elem, core.FlagNext)
			return nil
		}),
	)

	sending.
		DoFinally(func(s rx.SignalType) {
			ch <- struct{}{}
			<-ch
			select {
			case _, ok := <-ch:
				if ok {
					close(ch)
					dc.unregister(sid)
				}
			default:
			}
		}).
		SubscribeOn(scheduler.Parallel()).
		SubscribeWith(context.Background(), sub)

	<-mustSub
	return nil
}

func (dc *DuplexConnection) respondMetadataPush(input core.Frame) (err error) {
	defer func() {
		if e := recover(); e != nil {
			logger.Errorf("respond METADATA_PUSH failed: %s\n", e)
		}
	}()
	dc.responder.MetadataPush(input.(*framing.MetadataPushFrame))
	return
}

func (dc *DuplexConnection) onFrameFNF(frame core.Frame) error {
	receiving, ok := dc.doFragment(frame.(*framing.FireAndForgetFrame))
	if !ok {
		return nil
	}
	return dc.respondFNF(receiving)
}

func (dc *DuplexConnection) respondFNF(receiving fragmentation.HeaderAndPayload) (err error) {
	defer func() {
		if e := recover(); e != nil {
			logger.Errorf("respond FireAndForget failed: %s\n", e)
		}
	}()
	dc.responder.FireAndForget(receiving)
	return
}

func (dc *DuplexConnection) onFrameRequestStream(frame core.Frame) error {
	receiving, ok := dc.doFragment(frame.(*framing.RequestStreamFrame))
	if !ok {
		return nil
	}
	return dc.respondRequestStream(receiving)
}

func (dc *DuplexConnection) respondRequestStream(receiving fragmentation.HeaderAndPayload) error {
	sid := receiving.Header().StreamID()

	// execute request stream handler
	sending, err := func() (resp flux.Flux, err error) {
		defer func() {
			err = tryRecover(recover())
		}()
		resp = dc.responder.RequestStream(receiving)
		if resp == nil {
			err = framing.NewWriteableErrorFrame(sid, core.ErrorCodeApplicationError, unsupportedRequestStream)
		}
		return
	}()

	// send error with panic
	if err != nil {
		dc.writeError(sid, err)
		return nil
	}

	// seek n32
	var n32 int
	switch v := receiving.(type) {
	case *framing.RequestStreamFrame:
		n32 = int(v.InitialRequestN())
	case fragmentation.Joiner:
		n32 = int(v.First().(*framing.RequestStreamFrame).InitialRequestN())
	default:
		panic("unreachable")
	}

	sub := rx.NewSubscriber(
		rx.OnNext(func(elem payload.Payload) error {
			dc.sendPayload(sid, elem, core.FlagNext)
			return nil
		}),
		rx.OnSubscribe(func(s rx.Subscription) {
			dc.register(sid, requestStreamCallbackReverse{su: s})
			s.Request(n32)
		}),
		rx.OnError(func(e error) {
			dc.writeError(sid, e)
		}),
		rx.OnComplete(func() {
			dc.sendFrame(framing.NewPayloadFrame(sid, nil, nil, core.FlagComplete))
		}),
	)

	// async subscribe publisher
	sending.
		DoFinally(func(s rx.SignalType) {
			dc.unregister(sid)
		}).
		SubscribeOn(scheduler.Parallel()).
		SubscribeWith(context.Background(), sub)
	return nil
}

func (dc *DuplexConnection) writeError(sid uint32, e error) {
	// ignore sending error because current socket has been closed.
	if IsSocketClosedError(e) {
		return
	}
	switch err := e.(type) {
	case *framing.ErrorFrame:
		dc.sendFrame(err)
	case core.CustomError:
		dc.sendFrame(framing.NewWriteableErrorFrame(sid, err.ErrorCode(), err.ErrorData()))
	default:
		dc.sendFrame(framing.NewWriteableErrorFrame(sid, core.ErrorCodeApplicationError, []byte(e.Error())))
	}
}

// SetResponder sets a responder for current socket.
func (dc *DuplexConnection) SetResponder(responder Responder) {
	dc.responder = responder
}

func (dc *DuplexConnection) onFrameKeepalive(frame core.Frame) (err error) {
	f := frame.(*framing.KeepaliveFrame)
	if f.Header().Flag().Check(core.FlagRespond) {
		k := framing.NewKeepaliveFrame(f.LastReceivedPosition(), f.Data(), false)
		//f.SetHeader(framing.NewFrameHeader(0, framing.FrameTypeKeepalive))
		dc.sendFrame(k)
	}
	return
}

func (dc *DuplexConnection) onFrameCancel(frame core.Frame) (err error) {
	sid := frame.Header().StreamID()

	v, ok := dc.messages.Load(sid)
	if !ok {
		logger.Warnf("nothing cancelled: sid=%d\n", sid)
		return
	}

	switch vv := v.(type) {
	case requestResponseCallbackReverse:
		vv.su.Cancel()
	case requestStreamCallbackReverse:
		vv.su.Cancel()
	default:
		panic(fmt.Errorf("illegal cancel target: %v", vv))
	}

	if _, ok := dc.fragments.Load(sid); ok {
		dc.fragments.Delete(sid)
	}
	return
}

func (dc *DuplexConnection) onFrameError(input core.Frame) (err error) {
	f := input.(*framing.ErrorFrame)
	logger.Errorf("handle error frame: %s\n", f)
	sid := f.Header().StreamID()

	v, ok := dc.messages.Load(sid)
	if !ok {
		err = fmt.Errorf("invalid stream id: %d", sid)
		return
	}

	switch vv := v.(type) {
	case requestResponseCallback:
		vv.pc.Error(f)
	case requestStreamCallback:
		vv.pc.Error(f)
	case requestChannelCallback:
		vv.rcv.Error(f)
	default:
		panic(fmt.Errorf("illegal value for error: %v", vv))
	}
	return
}

func (dc *DuplexConnection) onFrameRequestN(input core.Frame) (err error) {
	f := input.(*framing.RequestNFrame)
	sid := f.Header().StreamID()
	v, ok := dc.messages.Load(sid)
	if !ok {
		if logger.IsDebugEnabled() {
			logger.Debugf("ignore non-exists RequestN: id=%d\n", sid)
		}
		return
	}
	n := ToIntRequestN(f.N())
	switch vv := v.(type) {
	case requestStreamCallbackReverse:
		vv.su.Request(n)
	case requestChannelCallback:
		vv.snd.Request(n)
	case requestChannelCallbackReverse:
		vv.snd.Request(n)
	default:
		panic(fmt.Errorf("illegal requestN for %+v", vv))
	}
	return
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

func (dc *DuplexConnection) onFramePayload(frame core.Frame) error {
	pl, ok := dc.doFragment(frame.(*framing.PayloadFrame))
	if !ok {
		return nil
	}
	h := pl.Header()
	t := h.Type()
	if t == core.FrameTypeRequestFNF {
		return dc.respondFNF(pl)
	}
	if t == core.FrameTypeRequestResponse {
		return dc.respondRequestResponse(pl)
	}
	if t == core.FrameTypeRequestStream {
		return dc.respondRequestStream(pl)
	}
	if t == core.FrameTypeRequestChannel {
		return dc.respondRequestChannel(pl)
	}

	sid := h.StreamID()
	v, ok := dc.messages.Load(sid)
	if !ok {
		logger.Warnf("unoccupied Payload(id=%d), maybe it has been canceled(server=%T)\n", sid, dc.sids)
		return nil
	}

	switch vv := v.(type) {
	case requestResponseCallback:
		vv.pc.Success(pl)
	case requestStreamCallback:
		fg := h.Flag()
		isNext := fg.Check(core.FlagNext)
		if isNext {
			vv.pc.Next(pl)
		}
		if fg.Check(core.FlagComplete) {
			// Release pure complete payload
			vv.pc.Complete()
		}
	case requestChannelCallback:
		fg := h.Flag()
		isNext := fg.Check(core.FlagNext)
		if isNext {
			vv.rcv.Next(pl)
		}
		if fg.Check(core.FlagComplete) {
			vv.rcv.Complete()
		}
	case requestChannelCallbackReverse:
		fg := h.Flag()
		isNext := fg.Check(core.FlagNext)
		if isNext {
			vv.rcv.Next(pl)
		}
		if fg.Check(core.FlagComplete) {
			vv.rcv.Complete()
		}
	default:
		panic(fmt.Errorf("illegal Payload for %v", vv))
	}
	return nil
}

func (dc *DuplexConnection) clearTransport() {
	dc.cond.L.Lock()
	dc.tp = nil
	dc.cond.L.Unlock()
}

// SetTransport sets a transport for current socket.
func (dc *DuplexConnection) SetTransport(tp *transport.Transport) {
	tp.RegisterHandler(transport.OnCancel, dc.onFrameCancel)
	tp.RegisterHandler(transport.OnError, dc.onFrameError)
	tp.RegisterHandler(transport.OnRequestN, dc.onFrameRequestN)
	tp.RegisterHandler(transport.OnPayload, dc.onFramePayload)
	tp.RegisterHandler(transport.OnKeepalive, dc.onFrameKeepalive)

	if dc.responder != nil {
		tp.RegisterHandler(transport.OnRequestResponse, dc.onFrameRequestResponse)
		tp.RegisterHandler(transport.OnMetadataPush, dc.respondMetadataPush)
		tp.RegisterHandler(transport.OnFireAndForget, dc.onFrameFNF)
		tp.RegisterHandler(transport.OnRequestStream, dc.onFrameRequestStream)
		tp.RegisterHandler(transport.OnRequestChannel, dc.onFrameRequestChannel)
	}

	dc.cond.L.Lock()
	dc.tp = tp
	dc.cond.Signal()
	dc.cond.L.Unlock()
}

func (dc *DuplexConnection) sendFrame(f core.WriteableFrame) {
	defer func() {
		if e := recover(); e != nil {
			logger.Warnf("send frame failed: %s\n", e)
		}
	}()
	dc.outs <- f
}

func (dc *DuplexConnection) sendPayload(
	sid uint32,
	sending payload.Payload,
	frameFlag core.FrameFlag,
) {
	d := sending.Data()
	m, _ := sending.Metadata()
	size := framing.CalcPayloadFrameSize(d, m)

	if !dc.shouldSplit(size) {
		dc.sendFrame(framing.NewWriteablePayloadFrame(sid, d, m, frameFlag))
		return
	}
	dc.doSplit(d, m, func(index int, result fragmentation.SplitResult) {
		flag := result.Flag
		if index == 0 {
			flag |= frameFlag
		} else {
			flag |= core.FlagNext
		}
		dc.sendFrame(framing.NewWriteablePayloadFrame(sid, result.Data, result.Metadata, flag))
	})
	return
}

func (dc *DuplexConnection) drainWithKeepaliveAndLease(leaseChan <-chan lease.Lease) (ok bool) {
	if len(dc.outs) > 0 {
		dc.drain(nil)
	}
	var out core.WriteableFrame
	select {
	case <-dc.keepaliver.C():
		ok = true
		out = framing.NewKeepaliveFrame(dc.counter.ReadBytes(), nil, true)
		if dc.tp != nil {
			err := dc.tp.Send(out, true)
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
		if dc.tp == nil {
			dc.outsPriority = append(dc.outsPriority, out)
		} else if err := dc.tp.Send(out, true); err != nil {
			logger.Errorf("send frame failed: %s\n", err.Error())
			dc.outsPriority = append(dc.outsPriority, out)
		}
	case out, ok = <-dc.outs:
		if !ok {
			return
		}
		if dc.tp == nil {
			dc.outsPriority = append(dc.outsPriority, out)
		} else if err := dc.tp.Send(out, true); err != nil {
			logger.Errorf("send frame failed: %s\n", err.Error())
			dc.outsPriority = append(dc.outsPriority, out)
		}
	}
	return
}

func (dc *DuplexConnection) drainWithKeepalive() (ok bool) {
	if len(dc.outs) > 0 {
		dc.drain(nil)
	}
	var out core.WriteableFrame

	select {
	case <-dc.keepaliver.C():
		ok = true
		out = framing.NewKeepaliveFrame(dc.counter.ReadBytes(), nil, true)
		if dc.tp != nil {
			err := dc.tp.Send(out, true)
			if err != nil {
				logger.Errorf("send keepalive frame failed: %s\n", err.Error())
			}
		}
	case out, ok = <-dc.outs:
		if !ok {
			return
		}
		if dc.tp == nil {
			dc.outsPriority = append(dc.outsPriority, out)
		} else if err := dc.tp.Send(out, true); err != nil {
			logger.Errorf("send frame failed: %s\n", err.Error())
			dc.outsPriority = append(dc.outsPriority, out)
		}
	}
	return
}

func (dc *DuplexConnection) drain(leaseChan <-chan lease.Lease) bool {
	var flush bool
	cycle := len(dc.outs)
	if cycle < 1 {
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
		case out, ok := <-dc.outs:
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

func (dc *DuplexConnection) drainOne(out core.WriteableFrame) (wrote bool) {
	if dc.tp == nil {
		dc.outsPriority = append(dc.outsPriority, out)
		return
	}
	err := dc.tp.Send(out, false)
	if err != nil {
		dc.outsPriority = append(dc.outsPriority, out)
		logger.Errorf("send frame failed: %s\n", err.Error())
		return
	}
	wrote = true
	return
}

func (dc *DuplexConnection) drainOutBack() {
	if len(dc.outsPriority) < 1 {
		return
	}
	defer func() {
		dc.outsPriority = dc.outsPriority[:0]
	}()
	if dc.tp == nil {
		return
	}
	var out core.WriteableFrame
	for i := range dc.outsPriority {
		out = dc.outsPriority[i]
		if err := dc.tp.Send(out, false); err != nil {
			out.Done()
			logger.Errorf("send frame failed: %v\n", err)
		}
	}
	if err := dc.tp.Flush(); err != nil {
		logger.Errorf("flush failed: %v\n", err)
	}
}

func (dc *DuplexConnection) loopWriteWithKeepaliver(ctx context.Context, leaseChan <-chan lease.Lease) error {
	for {
		if dc.tp == nil {
			dc.cond.L.Lock()
			dc.cond.Wait()
			dc.cond.L.Unlock()
		}

		select {
		case <-ctx.Done():
			dc.cleanOuts()
			return ctx.Err()
		default:
			// ignore
		}

		select {
		case <-dc.keepaliver.C():
			kf := framing.NewKeepaliveFrame(dc.counter.ReadBytes(), nil, true)
			if dc.tp != nil {
				err := dc.tp.Send(kf, true)
				if err != nil {
					logger.Errorf("send keepalive frame failed: %s\n", err.Error())
				}
			}
		default:
		}

		dc.drainOutBack()
		if leaseChan == nil && !dc.drainWithKeepalive() {
			break
		}
		if leaseChan != nil && !dc.drainWithKeepaliveAndLease(leaseChan) {
			break
		}
	}
	return nil
}

func (dc *DuplexConnection) cleanOuts() {
	dc.outsPriority = nil
}

func (dc *DuplexConnection) LoopWrite(ctx context.Context) error {
	defer close(dc.writeDone)

	var leaseChan chan lease.Lease
	if dc.leases != nil {
		leaseCtx, cancel := context.WithCancel(ctx)
		defer func() {
			cancel()
		}()
		if c, ok := dc.leases.Next(leaseCtx); ok {
			leaseChan = c
		}
	}

	if dc.keepaliver != nil {
		defer dc.keepaliver.Stop()
		return dc.loopWriteWithKeepaliver(ctx, leaseChan)
	}
	for {
		if dc.tp == nil {
			dc.cond.L.Lock()
			dc.cond.Wait()
			dc.cond.L.Unlock()
		}

		select {
		case <-ctx.Done():
			dc.cleanOuts()
			return ctx.Err()
		default:
		}

		dc.drainOutBack()
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
	dc.fragments.Delete(sid)
}

// NewServerDuplexConnection creates a new server-side DuplexConnection.
func NewServerDuplexConnection(mtu int, leases lease.Leases) *DuplexConnection {
	return newDuplexConnection(mtu, nil, &serverStreamIDs{}, leases)
}

// NewClientDuplexConnection creates a new client-side DuplexConnection.
func NewClientDuplexConnection(mtu int, keepaliveInterval time.Duration) *DuplexConnection {
	return newDuplexConnection(mtu, NewKeepaliver(keepaliveInterval), &clientStreamIDs{}, nil)
}

func newDuplexConnection(mtu int, ka *Keepaliver, sids StreamID, leases lease.Leases) *DuplexConnection {
	l := &sync.RWMutex{}
	return &DuplexConnection{
		l:               l,
		leases:          leases,
		outs:            make(chan core.WriteableFrame, _outChanSize),
		mtu:             mtu,
		messages:        &sync.Map{},
		sids:            sids,
		fragments:       &sync.Map{},
		writeDone:       make(chan struct{}),
		cond:            sync.NewCond(l),
		counter:         core.NewTrafficCounter(),
		keepaliver:      ka,
		singleScheduler: scheduler.NewSingle(64),
	}
}
