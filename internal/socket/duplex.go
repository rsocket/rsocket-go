package socket

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/jjeffcaii/reactor-go/scheduler"
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
	"go.uber.org/atomic"
)

const _outChanSize = 64

var (
	errSocketClosed            = errors.New("socket closed already")
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
	counter         *core.Counter
	tp              *transport.Transport
	outs            chan core.WriteableFrame
	outsPriority    []core.WriteableFrame
	responder       Responder
	messages        *sync.Map
	sids            StreamID
	mtu             int
	fragments       *sync.Map // common.U32Map // key=streamID, value=Joiner
	closed          *atomic.Bool
	done            chan struct{}
	keepaliver      *Keepaliver
	cond            *sync.Cond
	singleScheduler scheduler.Scheduler
	e               error
	leases          lease.Leases
}

// SetError sets error for current socket.
func (p *DuplexConnection) SetError(e error) {
	p.e = e
}

// GetError get the error set.
func (p *DuplexConnection) GetError() error {
	return p.e
}

func (p *DuplexConnection) nextStreamID() (sid uint32) {
	var lap1st bool
	for {
		// There's no required to check StreamID conflicts.
		sid, lap1st = p.sids.Next()
		if lap1st {
			return
		}
		_, ok := p.messages.Load(sid)
		if !ok {
			return
		}
	}
}

// Close close current socket.
func (p *DuplexConnection) Close() error {
	if !p.closed.CAS(false, true) {
		return nil
	}
	if p.keepaliver != nil {
		p.keepaliver.Stop()
	}
	_ = p.singleScheduler.(io.Closer).Close()
	close(p.outs)
	p.cond.L.Lock()
	p.cond.Broadcast()
	p.cond.L.Unlock()

	<-p.done

	if p.tp != nil {
		if p.e == nil {
			p.e = p.tp.Close()
		} else {
			_ = p.tp.Close()
		}
	}
	p.messages.Range(func(key, value interface{}) bool {
		if cc, ok := value.(callback); ok {
			if p.e == nil {
				go func() {
					cc.Close(errSocketClosed)
				}()
			} else {
				go func(e error) {
					cc.Close(e)
				}(p.e)
			}
		}
		return true
	})
	return p.e
}

// FireAndForget start a request of FireAndForget.
func (p *DuplexConnection) FireAndForget(sending payload.Payload) {
	data := sending.Data()
	size := core.FrameHeaderLen + len(sending.Data())
	m, ok := sending.Metadata()
	if ok {
		size += 3 + len(m)
	}
	sid := p.nextStreamID()
	if !p.shouldSplit(size) {
		p.sendFrame(framing.NewWriteableFireAndForgetFrame(sid, data, m, 0))
		return
	}
	p.doSplit(data, m, func(index int, result fragmentation.SplitResult) {
		var f core.WriteableFrame
		if index == 0 {
			f = framing.NewWriteableFireAndForgetFrame(sid, result.Data, result.Metadata, result.Flag)
		} else {
			f = framing.NewWriteablePayloadFrame(sid, result.Data, result.Metadata, result.Flag|core.FlagNext)
		}
		p.sendFrame(f)
	})
}

// MetadataPush start a request of MetadataPush.
func (p *DuplexConnection) MetadataPush(payload payload.Payload) {
	metadata, _ := payload.Metadata()
	p.sendFrame(framing.NewWriteableMetadataPushFrame(metadata))
}

// RequestResponse start a request of RequestResponse.
func (p *DuplexConnection) RequestResponse(pl payload.Payload) (mo mono.Mono) {
	sid := p.nextStreamID()
	resp := mono.CreateProcessor()

	p.register(sid, requestResponseCallback{pc: resp})

	data := pl.Data()
	metadata, _ := pl.Metadata()
	mo = resp.
		DoFinally(func(s rx.SignalType) {
			if s == rx.SignalCancel {
				p.sendFrame(framing.NewWriteableCancelFrame(sid))
			}
			p.unregister(sid)
		})

	p.singleScheduler.Worker().Do(func() {
		// sending...
		size := framing.CalcPayloadFrameSize(data, metadata)
		if !p.shouldSplit(size) {
			p.sendFrame(framing.NewWriteableRequestResponseFrame(sid, data, metadata, 0))
			return
		}
		p.doSplit(data, metadata, func(index int, result fragmentation.SplitResult) {
			var f core.WriteableFrame
			if index == 0 {
				f = framing.NewWriteableRequestResponseFrame(sid, result.Data, result.Metadata, result.Flag)
			} else {
				f = framing.NewWriteablePayloadFrame(sid, result.Data, result.Metadata, result.Flag|core.FlagNext)
			}
			p.sendFrame(f)
		})
	})
	return
}

// RequestStream start a request of RequestStream.
func (p *DuplexConnection) RequestStream(sending payload.Payload) (ret flux.Flux) {
	sid := p.nextStreamID()
	pc := flux.CreateProcessor()

	p.register(sid, requestStreamCallback{pc: pc})

	requested := make(chan struct{})

	ret = pc.
		DoFinally(func(sig rx.SignalType) {
			if sig == rx.SignalCancel {
				p.sendFrame(framing.NewWriteableCancelFrame(sid))
			}
			p.unregister(sid)
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
				p.sendFrame(frameN)
				<-frameN.DoneNotify()
				return
			}

			data := sending.Data()
			metadata, _ := sending.Metadata()

			size := framing.CalcPayloadFrameSize(data, metadata) + 4
			if !p.shouldSplit(size) {
				p.sendFrame(framing.NewWriteableRequestStreamFrame(sid, n32, data, metadata, 0))
				return
			}

			p.doSplitSkip(4, data, metadata, func(index int, result fragmentation.SplitResult) {
				var f core.WriteableFrame
				if index == 0 {
					f = framing.NewWriteableRequestStreamFrame(sid, n32, result.Data, result.Metadata, result.Flag)
				} else {
					f = framing.NewWriteablePayloadFrame(sid, result.Data, result.Metadata, result.Flag|core.FlagNext)
				}
				p.sendFrame(f)
			})
		})
	return
}

// RequestChannel start a request of RequestChannel.
func (p *DuplexConnection) RequestChannel(publisher rx.Publisher) (ret flux.Flux) {
	sid := p.nextStreamID()

	sending := publisher.(flux.Flux)
	receiving := flux.CreateProcessor()

	rcvRequested := make(chan struct{})

	ret = receiving.
		DoFinally(func(sig rx.SignalType) {
			p.unregister(sid)
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
				p.sendFrame(frameN)
				<-frameN.DoneNotify()
				return
			}

			sndRequested := make(chan struct{})
			sub := rx.NewSubscriber(
				rx.OnNext(func(item payload.Payload) {
					var newborn bool
					select {
					case <-sndRequested:
					default:
						newborn = true
						close(sndRequested)
					}
					if !newborn {
						p.sendPayload(sid, item, core.FlagNext)
						return
					}

					d := item.Data()
					m, _ := item.Metadata()
					size := framing.CalcPayloadFrameSize(d, m) + 4
					if !p.shouldSplit(size) {
						metadata, _ := item.Metadata()
						p.sendFrame(framing.NewWriteableRequestChannelFrame(sid, n32, item.Data(), metadata, core.FlagNext))
						return
					}

					p.doSplitSkip(4, d, m, func(index int, result fragmentation.SplitResult) {
						var f core.WriteableFrame
						if index == 0 {
							f = framing.NewWriteableRequestChannelFrame(sid, n32, result.Data, result.Metadata, result.Flag|core.FlagNext)
						} else {
							f = framing.NewWriteablePayloadFrame(sid, result.Data, result.Metadata, result.Flag|core.FlagNext)
						}
						p.sendFrame(f)
					})
				}),
				rx.OnSubscribe(func(s rx.Subscription) {
					p.register(sid, requestChannelCallback{rcv: receiving, snd: s})
					s.Request(1)
				}),
			)
			sending.
				DoFinally(func(sig rx.SignalType) {
					// TODO: handle cancel or error
					switch sig {
					case rx.SignalComplete:
						complete := framing.NewPayloadFrame(sid, nil, nil, core.FlagComplete)
						p.sendFrame(complete)
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

func (p *DuplexConnection) onFrameRequestResponse(frame core.Frame) error {
	// fragment
	receiving, ok := p.doFragment(frame.(*framing.RequestResponseFrame))
	if !ok {
		return nil
	}
	return p.respondRequestResponse(receiving)
}

func (p *DuplexConnection) respondRequestResponse(receiving fragmentation.HeaderAndPayload) error {
	sid := receiving.Header().StreamID()

	// 1. execute socket handler
	sending, err := func() (mono mono.Mono, err error) {
		defer func() {
			err = tryRecover(recover())
		}()
		mono = p.responder.RequestResponse(receiving)
		return
	}()
	// 2. sending error with panic
	if err != nil {
		p.writeError(sid, err)
		return nil
	}
	// 3. sending error with unsupported handler
	if sending == nil {
		p.writeError(sid, framing.NewWriteableErrorFrame(sid, core.ErrorCodeApplicationError, unsupportedRequestResponse))
		return nil
	}

	// 4. async subscribe publisher
	sub := rx.NewSubscriber(
		rx.OnNext(func(input payload.Payload) {
			p.sendPayload(sid, input, core.FlagNext|core.FlagComplete)
		}),
		rx.OnError(func(e error) {
			p.writeError(sid, e)
		}),
		rx.OnSubscribe(func(s rx.Subscription) {
			p.register(sid, requestResponseCallbackReverse{su: s})
			s.Request(rx.RequestMax)
		}),
	)
	sending.
		DoFinally(func(sig rx.SignalType) {
			p.unregister(sid)
		}).
		SubscribeOn(scheduler.Parallel()).
		SubscribeWith(context.Background(), sub)
	return nil
}

func (p *DuplexConnection) onFrameRequestChannel(input core.Frame) error {
	receiving, ok := p.doFragment(input.(*framing.RequestChannelFrame))
	if !ok {
		return nil
	}
	return p.respondRequestChannel(receiving)
}

func (p *DuplexConnection) respondRequestChannel(pl fragmentation.HeaderAndPayload) error {
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
					p.unregister(sid)
				}
			default:
			}
		}).
		DoOnRequest(func(n int) {
			frameN := framing.NewWriteableRequestNFrame(sid, ToUint32RequestN(n), 0)
			p.sendFrame(frameN)
			<-frameN.DoneNotify()
		})

	p.singleScheduler.Worker().Do(func() {
		receivingProcessor.Next(pl)
	})

	// TODO: if receiving == sending ???
	sending, err := func() (flux flux.Flux, err error) {
		defer func() {
			err = tryRecover(recover())
		}()
		flux = p.responder.RequestChannel(receiving)
		if flux == nil {
			err = framing.NewWriteableErrorFrame(sid, core.ErrorCodeApplicationError, unsupportedRequestChannel)
		}
		return
	}()

	if err != nil {
		p.writeError(sid, err)
		return nil
	}

	// Ensure registering message success before func end.
	mustSub := make(chan struct{})

	sub := rx.NewSubscriber(
		rx.OnError(func(e error) {
			p.writeError(sid, e)
		}),
		rx.OnComplete(func() {
			complete := framing.NewPayloadFrame(sid, nil, nil, core.FlagComplete)
			p.sendFrame(complete)
			<-complete.DoneNotify()
		}),
		rx.OnSubscribe(func(s rx.Subscription) {
			p.register(sid, requestChannelCallbackReverse{rcv: receivingProcessor, snd: s})
			close(mustSub)
			s.Request(initRequestN)
		}),
		rx.OnNext(func(elem payload.Payload) {
			p.sendPayload(sid, elem, core.FlagNext)
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
					p.unregister(sid)
				}
			default:
			}
		}).
		SubscribeOn(scheduler.Parallel()).
		SubscribeWith(context.Background(), sub)

	<-mustSub
	return nil
}

func (p *DuplexConnection) respondMetadataPush(input core.Frame) (err error) {
	defer func() {
		if e := recover(); e != nil {
			logger.Errorf("respond METADATA_PUSH failed: %s\n", e)
		}
	}()
	p.responder.MetadataPush(input.(*framing.MetadataPushFrame))
	return
}

func (p *DuplexConnection) onFrameFNF(frame core.Frame) error {
	receiving, ok := p.doFragment(frame.(*framing.FireAndForgetFrame))
	if !ok {
		return nil
	}
	return p.respondFNF(receiving)
}

func (p *DuplexConnection) respondFNF(receiving fragmentation.HeaderAndPayload) (err error) {
	defer func() {
		if e := recover(); e != nil {
			logger.Errorf("respond FireAndForget failed: %s\n", e)
		}
	}()
	p.responder.FireAndForget(receiving)
	return
}

func (p *DuplexConnection) onFrameRequestStream(frame core.Frame) error {
	receiving, ok := p.doFragment(frame.(*framing.RequestStreamFrame))
	if !ok {
		return nil
	}
	return p.respondRequestStream(receiving)
}

func (p *DuplexConnection) respondRequestStream(receiving fragmentation.HeaderAndPayload) error {
	sid := receiving.Header().StreamID()

	// execute request stream handler
	sending, err := func() (resp flux.Flux, err error) {
		defer func() {
			err = tryRecover(recover())
		}()
		resp = p.responder.RequestStream(receiving)
		if resp == nil {
			err = framing.NewWriteableErrorFrame(sid, core.ErrorCodeApplicationError, unsupportedRequestStream)
		}
		return
	}()

	// send error with panic
	if err != nil {
		p.writeError(sid, err)
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
		rx.OnNext(func(elem payload.Payload) {
			p.sendPayload(sid, elem, core.FlagNext)
		}),
		rx.OnSubscribe(func(s rx.Subscription) {
			p.register(sid, requestStreamCallbackReverse{su: s})
			s.Request(n32)
		}),
		rx.OnError(func(e error) {
			p.writeError(sid, e)
		}),
		rx.OnComplete(func() {
			p.sendFrame(framing.NewPayloadFrame(sid, nil, nil, core.FlagComplete))
		}),
	)

	// async subscribe publisher
	sending.
		DoFinally(func(s rx.SignalType) {
			p.unregister(sid)
		}).
		SubscribeOn(scheduler.Parallel()).
		SubscribeWith(context.Background(), sub)
	return nil
}

func (p *DuplexConnection) writeError(sid uint32, e error) {
	// ignore sending error because current socket has been closed.
	if IsSocketClosedError(e) {
		return
	}
	switch err := e.(type) {
	case *framing.ErrorFrame:
		p.sendFrame(err)
	case core.CustomError:
		p.sendFrame(framing.NewWriteableErrorFrame(sid, err.ErrorCode(), err.ErrorData()))
	default:
		p.sendFrame(framing.NewWriteableErrorFrame(sid, core.ErrorCodeApplicationError, []byte(e.Error())))
	}
}

// SetResponder sets a responder for current socket.
func (p *DuplexConnection) SetResponder(responder Responder) {
	p.responder = responder
}

func (p *DuplexConnection) onFrameKeepalive(frame core.Frame) (err error) {
	f := frame.(*framing.KeepaliveFrame)
	if f.Header().Flag().Check(core.FlagRespond) {
		k := framing.NewKeepaliveFrame(f.LastReceivedPosition(), f.Data(), false)
		//f.SetHeader(framing.NewFrameHeader(0, framing.FrameTypeKeepalive))
		p.sendFrame(k)
	}
	return
}

func (p *DuplexConnection) onFrameCancel(frame core.Frame) (err error) {
	sid := frame.Header().StreamID()

	v, ok := p.messages.Load(sid)
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

	if _, ok := p.fragments.Load(sid); ok {
		p.fragments.Delete(sid)
	}
	return
}

func (p *DuplexConnection) onFrameError(input core.Frame) (err error) {
	f := input.(*framing.ErrorFrame)
	logger.Errorf("handle error frame: %s\n", f)
	sid := f.Header().StreamID()

	v, ok := p.messages.Load(sid)
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

func (p *DuplexConnection) onFrameRequestN(input core.Frame) (err error) {
	f := input.(*framing.RequestNFrame)
	sid := f.Header().StreamID()
	v, ok := p.messages.Load(sid)
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

func (p *DuplexConnection) doFragment(input fragmentation.HeaderAndPayload) (out fragmentation.HeaderAndPayload, ok bool) {
	h := input.Header()
	sid := h.StreamID()
	v, exist := p.fragments.Load(sid)
	if exist {
		joiner := v.(fragmentation.Joiner)
		ok = joiner.Push(input)
		if ok {
			p.fragments.Delete(sid)
			out = joiner
		}
		return
	}
	ok = !h.Flag().Check(core.FlagFollow)
	if ok {
		out = input
		return
	}
	p.fragments.Store(sid, fragmentation.NewJoiner(input))
	return
}

func (p *DuplexConnection) onFramePayload(frame core.Frame) error {
	pl, ok := p.doFragment(frame.(*framing.PayloadFrame))
	if !ok {
		return nil
	}
	h := pl.Header()
	t := h.Type()
	if t == core.FrameTypeRequestFNF {
		return p.respondFNF(pl)
	}
	if t == core.FrameTypeRequestResponse {
		return p.respondRequestResponse(pl)
	}
	if t == core.FrameTypeRequestStream {
		return p.respondRequestStream(pl)
	}
	if t == core.FrameTypeRequestChannel {
		return p.respondRequestChannel(pl)
	}

	sid := h.StreamID()
	v, ok := p.messages.Load(sid)
	if !ok {
		logger.Warnf("unoccupied Payload(id=%d), maybe it has been canceled(server=%T)\n", sid, p.sids)
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

func (p *DuplexConnection) clearTransport() {
	p.cond.L.Lock()
	p.tp = nil
	p.cond.L.Unlock()
}

// SetTransport sets a transport for current socket.
func (p *DuplexConnection) SetTransport(tp *transport.Transport) {
	tp.RegisterHandler(transport.OnCancel, p.onFrameCancel)
	tp.RegisterHandler(transport.OnError, p.onFrameError)
	tp.RegisterHandler(transport.OnRequestN, p.onFrameRequestN)
	tp.RegisterHandler(transport.OnPayload, p.onFramePayload)
	tp.RegisterHandler(transport.OnKeepalive, p.onFrameKeepalive)

	if p.responder != nil {
		tp.RegisterHandler(transport.OnRequestResponse, p.onFrameRequestResponse)
		tp.RegisterHandler(transport.OnMetadataPush, p.respondMetadataPush)
		tp.RegisterHandler(transport.OnFireAndForget, p.onFrameFNF)
		tp.RegisterHandler(transport.OnRequestStream, p.onFrameRequestStream)
		tp.RegisterHandler(transport.OnRequestChannel, p.onFrameRequestChannel)
	}

	p.cond.L.Lock()
	p.tp = tp
	p.cond.Signal()
	p.cond.L.Unlock()
}

func (p *DuplexConnection) sendFrame(f core.WriteableFrame) {
	defer func() {
		if e := recover(); e != nil {
			logger.Warnf("send frame failed: %s\n", e)
		}
	}()
	p.outs <- f
}

func (p *DuplexConnection) sendPayload(
	sid uint32,
	sending payload.Payload,
	frameFlag core.FrameFlag,
) {
	d := sending.Data()
	m, _ := sending.Metadata()
	size := framing.CalcPayloadFrameSize(d, m)

	if !p.shouldSplit(size) {
		p.sendFrame(framing.NewWriteablePayloadFrame(sid, d, m, frameFlag))
		return
	}
	p.doSplit(d, m, func(index int, result fragmentation.SplitResult) {
		flag := result.Flag
		if index == 0 {
			flag |= frameFlag
		} else {
			flag |= core.FlagNext
		}
		p.sendFrame(framing.NewWriteablePayloadFrame(sid, result.Data, result.Metadata, flag))
	})
}

func (p *DuplexConnection) drainWithKeepaliveAndLease(leaseChan <-chan lease.Lease) (ok bool) {
	if len(p.outs) > 0 {
		p.drain(nil)
	}
	var out core.WriteableFrame
	select {
	case <-p.keepaliver.C():
		ok = true
		out = framing.NewKeepaliveFrame(p.counter.ReadBytes(), nil, true)
		if p.tp != nil {
			err := p.tp.Send(out, true)
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
		if p.tp == nil {
			p.outsPriority = append(p.outsPriority, out)
		} else if err := p.tp.Send(out, true); err != nil {
			logger.Errorf("send frame failed: %s\n", err.Error())
			p.outsPriority = append(p.outsPriority, out)
		}
	case out, ok = <-p.outs:
		if !ok {
			return
		}
		if p.tp == nil {
			p.outsPriority = append(p.outsPriority, out)
		} else if err := p.tp.Send(out, true); err != nil {
			logger.Errorf("send frame failed: %s\n", err.Error())
			p.outsPriority = append(p.outsPriority, out)
		}
	}
	return
}

func (p *DuplexConnection) drainWithKeepalive() (ok bool) {
	if len(p.outs) > 0 {
		p.drain(nil)
	}
	var out core.WriteableFrame

	select {
	case <-p.keepaliver.C():
		ok = true
		out = framing.NewKeepaliveFrame(p.counter.ReadBytes(), nil, true)
		if p.tp != nil {
			err := p.tp.Send(out, true)
			if err != nil {
				logger.Errorf("send keepalive frame failed: %s\n", err.Error())
			}
		}
	case out, ok = <-p.outs:
		if !ok {
			return
		}
		if p.tp == nil {
			p.outsPriority = append(p.outsPriority, out)
		} else if err := p.tp.Send(out, true); err != nil {
			logger.Errorf("send frame failed: %s\n", err.Error())
			p.outsPriority = append(p.outsPriority, out)
		}
	}
	return
}

func (p *DuplexConnection) drain(leaseChan <-chan lease.Lease) bool {
	var flush bool
	cycle := len(p.outs)
	if cycle < 1 {
		cycle = 1
	}
	for i := 0; i < cycle; i++ {
		select {
		case next, ok := <-leaseChan:
			if !ok {
				return false
			}
			if p.drainOne(framing.NewWriteableLeaseFrame(next.TimeToLive, next.NumberOfRequests, next.Metadata)) {
				flush = true
			}
		case out, ok := <-p.outs:
			if !ok {
				return false
			}
			if p.drainOne(out) {
				flush = true
			}
		}
	}
	if flush {
		if err := p.tp.Flush(); err != nil {
			logger.Errorf("flush failed: %v\n", err)
		}
	}
	return true
}

func (p *DuplexConnection) drainOne(out core.WriteableFrame) (wrote bool) {
	if p.tp == nil {
		p.outsPriority = append(p.outsPriority, out)
		return
	}
	err := p.tp.Send(out, false)
	if err != nil {
		p.outsPriority = append(p.outsPriority, out)
		logger.Errorf("send frame failed: %s\n", err.Error())
		return
	}
	wrote = true
	return
}

func (p *DuplexConnection) drainOutBack() {
	if len(p.outsPriority) < 1 {
		return
	}
	defer func() {
		p.outsPriority = p.outsPriority[:0]
	}()
	if p.tp == nil {
		return
	}
	var out core.WriteableFrame
	for i := range p.outsPriority {
		out = p.outsPriority[i]
		if err := p.tp.Send(out, false); err != nil {
			out.Done()
			logger.Errorf("send frame failed: %v\n", err)
		}
	}
	if err := p.tp.Flush(); err != nil {
		logger.Errorf("flush failed: %v\n", err)
	}
}

func (p *DuplexConnection) loopWriteWithKeepaliver(ctx context.Context, leaseChan <-chan lease.Lease) error {
	for {
		if p.tp == nil {
			p.cond.L.Lock()
			p.cond.Wait()
			p.cond.L.Unlock()
		}

		select {
		case <-ctx.Done():
			p.cleanOuts()
			return ctx.Err()
		default:
			// ignore
		}

		select {
		case <-p.keepaliver.C():
			kf := framing.NewKeepaliveFrame(p.counter.ReadBytes(), nil, true)
			if p.tp != nil {
				err := p.tp.Send(kf, true)
				if err != nil {
					logger.Errorf("send keepalive frame failed: %s\n", err.Error())
				}
			}
		default:
		}

		p.drainOutBack()
		if leaseChan == nil && !p.drainWithKeepalive() {
			break
		}
		if leaseChan != nil && !p.drainWithKeepaliveAndLease(leaseChan) {
			break
		}
	}
	return nil
}

func (p *DuplexConnection) cleanOuts() {
	p.outsPriority = nil
}

func (p *DuplexConnection) LoopWrite(ctx context.Context) error {
	defer close(p.done)

	var leaseChan chan lease.Lease
	if p.leases != nil {
		leaseCtx, cancel := context.WithCancel(ctx)
		defer func() {
			cancel()
		}()
		if c, ok := p.leases.Next(leaseCtx); ok {
			leaseChan = c
		}
	}

	if p.keepaliver != nil {
		defer p.keepaliver.Stop()
		return p.loopWriteWithKeepaliver(ctx, leaseChan)
	}
	for {
		if p.tp == nil {
			p.cond.L.Lock()
			p.cond.Wait()
			p.cond.L.Unlock()
		}

		select {
		case <-ctx.Done():
			p.cleanOuts()
			return ctx.Err()
		default:
		}

		p.drainOutBack()
		if !p.drain(leaseChan) {
			break
		}
	}
	return nil
}

func (p *DuplexConnection) doSplit(data, metadata []byte, handler fragmentation.HandleSplitResult) {
	fragmentation.Split(p.mtu, data, metadata, handler)
}

func (p *DuplexConnection) doSplitSkip(skip int, data, metadata []byte, handler fragmentation.HandleSplitResult) {
	fragmentation.SplitSkip(p.mtu, skip, data, metadata, handler)
}

func (p *DuplexConnection) shouldSplit(size int) bool {
	return size > p.mtu
}

func (p *DuplexConnection) register(sid uint32, msg interface{}) {
	p.messages.Store(sid, msg)
}

func (p *DuplexConnection) unregister(sid uint32) {
	p.messages.Delete(sid)
	p.fragments.Delete(sid)
}

// NewServerDuplexConnection creates a new server-side DuplexConnection.
func NewServerDuplexConnection(mtu int, leases lease.Leases) *DuplexConnection {
	return &DuplexConnection{
		closed:          atomic.NewBool(false),
		leases:          leases,
		outs:            make(chan core.WriteableFrame, _outChanSize),
		mtu:             mtu,
		messages:        &sync.Map{},
		sids:            &serverStreamIDs{},
		fragments:       &sync.Map{},
		done:            make(chan struct{}),
		cond:            sync.NewCond(&sync.Mutex{}),
		counter:         core.NewCounter(),
		singleScheduler: scheduler.NewSingle(64),
	}
}

// NewClientDuplexConnection creates a new client-side DuplexConnection.
func NewClientDuplexConnection(
	mtu int,
	keepaliveInterval time.Duration,
) (s *DuplexConnection) {
	ka := NewKeepaliver(keepaliveInterval)
	s = &DuplexConnection{
		closed:          atomic.NewBool(false),
		outs:            make(chan core.WriteableFrame, _outChanSize),
		mtu:             mtu,
		messages:        &sync.Map{},
		sids:            &clientStreamIDs{},
		fragments:       &sync.Map{},
		done:            make(chan struct{}),
		cond:            sync.NewCond(&sync.Mutex{}),
		counter:         core.NewCounter(),
		keepaliver:      ka,
		singleScheduler: scheduler.NewSingle(64),
	}
	return
}
