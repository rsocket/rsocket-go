package socket

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/fragmentation"
	"github.com/rsocket/rsocket-go/internal/framing"
	"github.com/rsocket/rsocket-go/internal/transport"
	"github.com/rsocket/rsocket-go/lease"
	"github.com/rsocket/rsocket-go/logger"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/rsocket/rsocket-go/rx/mono"
	"go.uber.org/atomic"
)

const outsSize = 64

var (
	errSocketClosed            = errors.New("socket closed already")
	unsupportedRequestStream   = []byte("Request-Stream not implemented.")
	unsupportedRequestResponse = []byte("Request-Response not implemented.")
	unsupportedRequestChannel  = []byte("Request-Channel not implemented.")
)

// DuplexRSocket represents a socket of RSocket which can be a requester or a responder.
type DuplexRSocket struct {
	counter         *transport.Counter
	tp              *transport.Transport
	outs            chan framing.Frame
	outsPriority    []framing.Frame
	responder       Responder
	messages        *u32map
	sids            streamIDs
	mtu             int
	fragments       *u32map // key=streamID, value=Joiner
	closed          *atomic.Bool
	done            chan struct{}
	keepaliver      *keepaliver
	cond            *sync.Cond
	singleScheduler scheduler.Scheduler
	e               error
	leases          lease.Leases
}

// SetError sets error for current socket.
func (p *DuplexRSocket) SetError(e error) {
	p.e = e
}

func (p *DuplexRSocket) nextStreamID() (sid uint32) {
	var lap1st bool
	for {
		// There's no necessery to check StreamID conflicts.
		sid, lap1st = p.sids.next()
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
func (p *DuplexRSocket) Close() error {
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

	_ = p.fragments.Close()
	<-p.done

	if p.tp != nil {
		if p.e == nil {
			p.e = p.tp.Close()
		} else {
			_ = p.tp.Close()
		}
	}

	p.fragments.Range(func(key uint32, value interface{}) bool {
		return true
	})
	_ = p.fragments.Close()

	p.messages.Range(func(key uint32, value interface{}) bool {
		if cc, ok := value.(closerWithError); ok {
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
	_ = p.messages.Close()
	return p.e
}

// FireAndForget start a request of FireAndForget.
func (p *DuplexRSocket) FireAndForget(sending payload.Payload) {
	data := sending.Data()
	size := framing.HeaderLen + len(sending.Data())
	m, ok := sending.Metadata()
	if ok {
		size += 3 + len(m)
	}
	sid := p.nextStreamID()
	if !p.shouldSplit(size) {
		p.sendFrame(framing.NewFrameFNF(sid, data, m))
		return
	}
	p.doSplit(data, m, func(idx int, fg framing.FrameFlag, body *common.ByteBuff) {
		var f framing.Frame
		if idx == 0 {
			h := framing.NewFrameHeader(sid, framing.FrameTypeRequestFNF, fg)
			f = &framing.FrameFNF{
				BaseFrame: framing.NewBaseFrame(h, body),
			}
		} else {
			h := framing.NewFrameHeader(sid, framing.FrameTypePayload, fg|framing.FlagNext)
			f = &framing.FramePayload{
				BaseFrame: framing.NewBaseFrame(h, body),
			}
		}
		p.sendFrame(f)
	})
}

// MetadataPush start a request of MetadataPush.
func (p *DuplexRSocket) MetadataPush(payload payload.Payload) {
	metadata, _ := payload.Metadata()
	p.sendFrame(framing.NewFrameMetadataPush(metadata))
}

// RequestResponse start a request of RequestResponse.
func (p *DuplexRSocket) RequestResponse(pl payload.Payload) (mo mono.Mono) {
	sid := p.nextStreamID()
	resp := mono.CreateProcessor()

	p.register(sid, reqRR{pc: resp})

	data := pl.Data()
	metadata, _ := pl.Metadata()
	mo = resp.
		DoFinally(func(s rx.SignalType) {
			if s == rx.SignalCancel {
				p.sendFrame(framing.NewFrameCancel(sid))
			}
			p.unregister(sid)
		})

	p.singleScheduler.Worker().Do(func() {
		// sending...
		size := framing.CalcPayloadFrameSize(data, metadata)
		if !p.shouldSplit(size) {
			p.sendFrame(framing.NewFrameRequestResponse(sid, data, metadata))
			return
		}
		p.doSplit(data, metadata, func(idx int, fg framing.FrameFlag, body *common.ByteBuff) {
			var f framing.Frame
			if idx == 0 {
				h := framing.NewFrameHeader(sid, framing.FrameTypeRequestResponse, fg)
				f = &framing.FrameRequestResponse{
					BaseFrame: framing.NewBaseFrame(h, body),
				}
			} else {
				h := framing.NewFrameHeader(sid, framing.FrameTypePayload, fg|framing.FlagNext)
				f = &framing.FramePayload{
					BaseFrame: framing.NewBaseFrame(h, body),
				}
			}
			p.sendFrame(f)
		})
	})
	return
}

// RequestStream start a request of RequestStream.
func (p *DuplexRSocket) RequestStream(sending payload.Payload) (ret flux.Flux) {
	sid := p.nextStreamID()
	pc := flux.CreateProcessor()

	p.register(sid, reqRS{pc: pc})

	requested := make(chan struct{})

	ret = pc.
		DoFinally(func(sig rx.SignalType) {
			if sig == rx.SignalCancel {
				p.sendFrame(framing.NewFrameCancel(sid))
			}
			p.unregister(sid)
		}).
		DoOnRequest(func(n int) {
			n32 := toU32N(n)

			var newborn bool
			select {
			case <-requested:
			default:
				newborn = true
				close(requested)
			}

			if !newborn {
				frameN := framing.NewFrameRequestN(sid, n32)
				p.sendFrame(frameN)
				<-frameN.DoneNotify()
				return
			}

			data := sending.Data()
			metadata, _ := sending.Metadata()

			size := framing.CalcPayloadFrameSize(data, metadata) + 4
			if !p.shouldSplit(size) {
				p.sendFrame(framing.NewFrameRequestStream(sid, n32, data, metadata))
				return
			}
			p.doSplitSkip(4, data, metadata, func(idx int, fg framing.FrameFlag, body *common.ByteBuff) {
				var f framing.Frame
				if idx == 0 {
					h := framing.NewFrameHeader(sid, framing.FrameTypeRequestStream, fg)
					// write init RequestN
					binary.BigEndian.PutUint32(body.Bytes(), n32)
					f = &framing.FrameRequestStream{
						BaseFrame: framing.NewBaseFrame(h, body),
					}
				} else {
					h := framing.NewFrameHeader(sid, framing.FrameTypePayload, fg|framing.FlagNext)
					f = &framing.FramePayload{
						BaseFrame: framing.NewBaseFrame(h, body),
					}
				}
				p.sendFrame(f)
			})
		})
	return
}

// RequestChannel start a request of RequestChannel.
func (p *DuplexRSocket) RequestChannel(publisher rx.Publisher) (ret flux.Flux) {
	sid := p.nextStreamID()

	sending := publisher.(flux.Flux)
	receiving := flux.CreateProcessor()

	rcvRequested := make(chan struct{})

	ret = receiving.
		DoFinally(func(sig rx.SignalType) {
			p.unregister(sid)
		}).
		DoOnRequest(func(n int) {
			n32 := toU32N(n)
			var newborn bool
			select {
			case <-rcvRequested:
			default:
				newborn = true
				close(rcvRequested)
			}
			if !newborn {
				frameN := framing.NewFrameRequestN(sid, n32)
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
						p.sendPayload(sid, item, framing.FlagNext)
						return
					}

					d := item.Data()
					m, _ := item.Metadata()
					size := framing.CalcPayloadFrameSize(d, m) + 4
					if !p.shouldSplit(size) {
						metadata, _ := item.Metadata()
						p.sendFrame(framing.NewFrameRequestChannel(sid, n32, item.Data(), metadata, framing.FlagNext))
						return
					}
					p.doSplitSkip(4, d, m, func(idx int, fg framing.FrameFlag, body *common.ByteBuff) {
						var f framing.Frame
						if idx == 0 {
							h := framing.NewFrameHeader(sid, framing.FrameTypeRequestChannel, fg|framing.FlagNext)
							// write init RequestN
							binary.BigEndian.PutUint32(body.Bytes(), n32)
							f = &framing.FrameRequestChannel{
								BaseFrame: framing.NewBaseFrame(h, body),
							}
						} else {
							h := framing.NewFrameHeader(sid, framing.FrameTypePayload, fg|framing.FlagNext)
							f = &framing.FramePayload{
								BaseFrame: framing.NewBaseFrame(h, body),
							}
						}
						p.sendFrame(f)
					})
				}),
				rx.OnSubscribe(func(s rx.Subscription) {
					p.register(sid, reqRC{rcv: receiving, snd: s})
					s.Request(1)
				}),
			)
			sending.
				DoFinally(func(sig rx.SignalType) {
					// TODO: handle cancel or error
					switch sig {
					case rx.SignalComplete:
						complete := framing.NewFramePayload(sid, nil, nil, framing.FlagComplete)
						p.sendFrame(complete)
						<-complete.DoneNotify()
					default:
						panic(fmt.Errorf("unsupported sending channel signal: %s", sig))
					}
				}).
				SubscribeOn(scheduler.Elastic()).
				SubscribeWith(context.Background(), sub)
		})
	return ret
}

func (p *DuplexRSocket) onFrameRequestResponse(frame framing.Frame) error {
	// fragment
	receiving, ok := p.doFragment(frame.(*framing.FrameRequestResponse))
	if !ok {
		return nil
	}
	return p.respondRequestResponse(receiving)
}

func (p *DuplexRSocket) respondRequestResponse(receiving fragmentation.HeaderAndPayload) error {
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
		p.writeError(sid, framing.NewFrameError(sid, common.ErrorCodeApplicationError, unsupportedRequestResponse))
		return nil
	}

	// 4. async subscribe publisher
	sub := rx.NewSubscriber(
		rx.OnNext(func(input payload.Payload) {
			p.sendPayload(sid, input, framing.FlagNext|framing.FlagComplete)
		}),
		rx.OnError(func(e error) {
			p.writeError(sid, e)
		}),
		rx.OnSubscribe(func(s rx.Subscription) {
			p.register(sid, resRR{su: s})
			s.Request(rx.RequestMax)
		}),
	)
	sending.
		DoFinally(func(sig rx.SignalType) {
			p.unregister(sid)
		}).
		SubscribeOn(scheduler.Elastic()).
		SubscribeWith(context.Background(), sub)
	return nil
}

func (p *DuplexRSocket) onFrameRequestChannel(input framing.Frame) error {
	receiving, ok := p.doFragment(input.(*framing.FrameRequestChannel))
	if !ok {
		return nil
	}
	return p.respondRequestChannel(receiving)
}

func (p *DuplexRSocket) respondRequestChannel(pl fragmentation.HeaderAndPayload) error {
	// seek initRequestN
	var initRequestN int
	switch v := pl.(type) {
	case *framing.FrameRequestChannel:
		initRequestN = toIntN(v.InitialRequestN())
	case fragmentation.Joiner:
		initRequestN = toIntN(v.First().(*framing.FrameRequestChannel).InitialRequestN())
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
			frameN := framing.NewFrameRequestN(sid, toU32N(n))
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
			err = framing.NewFrameError(sid, common.ErrorCodeApplicationError, unsupportedRequestChannel)
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
			complete := framing.NewFramePayload(sid, nil, nil, framing.FlagComplete)
			p.sendFrame(complete)
			<-complete.DoneNotify()
		}),
		rx.OnSubscribe(func(s rx.Subscription) {
			p.register(sid, resRC{rcv: receivingProcessor, snd: s})
			close(mustSub)
			s.Request(initRequestN)
		}),
		rx.OnNext(func(elem payload.Payload) {
			p.sendPayload(sid, elem, framing.FlagNext)
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
		SubscribeOn(scheduler.Elastic()).
		SubscribeWith(context.Background(), sub)

	<-mustSub
	return nil
}

func (p *DuplexRSocket) respondMetadataPush(input framing.Frame) (err error) {
	defer func() {
		if e := recover(); e != nil {
			logger.Errorf("respond METADATA_PUSH failed: %s\n", e)
		}
	}()
	p.responder.MetadataPush(input.(*framing.FrameMetadataPush))
	return
}

func (p *DuplexRSocket) onFrameFNF(frame framing.Frame) error {
	receiving, ok := p.doFragment(frame.(*framing.FrameFNF))
	if !ok {
		return nil
	}
	return p.respondFNF(receiving)
}

func (p *DuplexRSocket) respondFNF(receiving fragmentation.HeaderAndPayload) (err error) {
	defer func() {
		if e := recover(); e != nil {
			logger.Errorf("respond FireAndForget failed: %s\n", e)
		}
	}()
	p.responder.FireAndForget(receiving)
	return
}

func (p *DuplexRSocket) onFrameRequestStream(frame framing.Frame) error {
	receiving, ok := p.doFragment(frame.(*framing.FrameRequestStream))
	if !ok {
		return nil
	}
	return p.respondRequestStream(receiving)
}

func (p *DuplexRSocket) respondRequestStream(receiving fragmentation.HeaderAndPayload) error {
	sid := receiving.Header().StreamID()

	// execute request stream handler
	sending, err := func() (resp flux.Flux, err error) {
		defer func() {
			err = tryRecover(recover())
		}()
		resp = p.responder.RequestStream(receiving)
		if resp == nil {
			err = framing.NewFrameError(sid, common.ErrorCodeApplicationError, unsupportedRequestStream)
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
	case *framing.FrameRequestStream:
		n32 = int(v.InitialRequestN())
	case fragmentation.Joiner:
		n32 = int(v.First().(*framing.FrameRequestStream).InitialRequestN())
	default:
		panic("unreachable")
	}

	sub := rx.NewSubscriber(
		rx.OnNext(func(elem payload.Payload) {
			p.sendPayload(sid, elem, framing.FlagNext)
		}),
		rx.OnSubscribe(func(s rx.Subscription) {
			p.register(sid, resRS{su: s})
			s.Request(n32)
		}),
		rx.OnError(func(e error) {
			p.writeError(sid, e)
		}),
		rx.OnComplete(func() {
			p.sendFrame(framing.NewFramePayload(sid, nil, nil, framing.FlagComplete))
		}),
	)

	// async subscribe publisher
	sending.
		DoFinally(func(s rx.SignalType) {
			p.unregister(sid)
		}).
		SubscribeOn(scheduler.Elastic()).
		SubscribeWith(context.Background(), sub)
	return nil
}

func (p *DuplexRSocket) writeError(sid uint32, e error) {
	// ignore sending error because current socket has been closed.
	if e == errSocketClosed {
		return
	}
	switch err := e.(type) {
	case *framing.FrameError:
		p.sendFrame(err)
	case common.CustomError:
		p.sendFrame(framing.NewFrameError(sid, err.ErrorCode(), err.ErrorData()))
	default:
		p.sendFrame(framing.NewFrameError(sid, common.ErrorCodeApplicationError, []byte(e.Error())))
	}
}

// SetResponder sets a responder for current socket.
func (p *DuplexRSocket) SetResponder(responder Responder) {
	p.responder = responder
}

func (p *DuplexRSocket) onFrameKeepalive(frame framing.Frame) (err error) {
	f := frame.(*framing.FrameKeepalive)
	if f.Header().Flag().Check(framing.FlagRespond) {
		f.SetHeader(framing.NewFrameHeader(0, framing.FrameTypeKeepalive))
		p.sendFrame(f)
	}
	return
}

func (p *DuplexRSocket) onFrameCancel(frame framing.Frame) (err error) {
	sid := frame.Header().StreamID()

	v, ok := p.messages.Load(sid)
	if !ok {
		logger.Warnf("nothing cancelled: sid=%d\n", sid)
		return
	}

	switch vv := v.(type) {
	case resRR:
		vv.su.Cancel()
	case resRS:
		vv.su.Cancel()
	default:
		panic(fmt.Errorf("illegal cancel target: %v", vv))
	}

	if _, ok := p.fragments.Load(sid); ok {
		p.fragments.Delete(sid)
	}
	return
}

func (p *DuplexRSocket) onFrameError(input framing.Frame) (err error) {
	f := input.(*framing.FrameError)
	logger.Errorf("handle error frame: %s\n", f)
	sid := f.Header().StreamID()

	v, ok := p.messages.Load(sid)
	if !ok {
		err = fmt.Errorf("invalid stream id: %d", sid)
		return
	}

	switch vv := v.(type) {
	case reqRR:
		vv.pc.Error(f)
	case reqRS:
		vv.pc.Error(f)
	case reqRC:
		vv.rcv.Error(f)
	default:
		panic(fmt.Errorf("illegal value for error: %v", vv))
	}
	return
}

func (p *DuplexRSocket) onFrameRequestN(input framing.Frame) (err error) {
	f := input.(*framing.FrameRequestN)
	sid := f.Header().StreamID()
	v, ok := p.messages.Load(sid)
	if !ok {
		if logger.IsDebugEnabled() {
			logger.Debugf("ignore non-exists RequestN: id=%d\n", sid)
		}
		return
	}
	n := toIntN(f.N())
	switch vv := v.(type) {
	case resRS:
		vv.su.Request(n)
	case reqRC:
		vv.snd.Request(n)
	case resRC:
		vv.snd.Request(n)
	default:
		panic(fmt.Errorf("illegal requestN for %+v", vv))
	}
	return
}

func (p *DuplexRSocket) doFragment(input fragmentation.HeaderAndPayload) (out fragmentation.HeaderAndPayload, ok bool) {
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
	ok = !h.Flag().Check(framing.FlagFollow)
	if ok {
		out = input
		return
	}
	p.fragments.Store(sid, fragmentation.NewJoiner(input))
	return
}

func (p *DuplexRSocket) onFramePayload(frame framing.Frame) error {
	pl, ok := p.doFragment(frame.(*framing.FramePayload))
	if !ok {
		return nil
	}
	h := pl.Header()
	t := h.Type()
	if t == framing.FrameTypeRequestFNF {
		return p.respondFNF(pl)
	}
	if t == framing.FrameTypeRequestResponse {
		return p.respondRequestResponse(pl)
	}
	if t == framing.FrameTypeRequestStream {
		return p.respondRequestStream(pl)
	}
	if t == framing.FrameTypeRequestChannel {
		return p.respondRequestChannel(pl)
	}

	sid := h.StreamID()
	v, ok := p.messages.Load(sid)
	if !ok {
		logger.Warnf("unoccupied Payload(id=%d), maybe it has been canceled(server=%T)\n", sid, p.sids)
		return nil
	}

	switch vv := v.(type) {
	case reqRR:
		vv.pc.Success(pl)
	case reqRS:
		fg := h.Flag()
		isNext := fg.Check(framing.FlagNext)
		if isNext {
			vv.pc.Next(pl)
		}
		if fg.Check(framing.FlagComplete) {
			// Release pure complete payload
			vv.pc.Complete()
		}
	case reqRC:
		fg := h.Flag()
		isNext := fg.Check(framing.FlagNext)
		if isNext {
			vv.rcv.Next(pl)
		}
		if fg.Check(framing.FlagComplete) {
			vv.rcv.Complete()
		}
	case resRC:
		fg := h.Flag()
		isNext := fg.Check(framing.FlagNext)
		if isNext {
			vv.rcv.Next(pl)
		}
		if fg.Check(framing.FlagComplete) {
			vv.rcv.Complete()
		}
	default:
		panic(fmt.Errorf("illegal Payload for %v", vv))
	}
	return nil
}

func (p *DuplexRSocket) clearTransport() {
	p.cond.L.Lock()
	p.tp = nil
	p.cond.L.Unlock()
}

// SetTransport sets a transport for current socket.
func (p *DuplexRSocket) SetTransport(tp *transport.Transport) {
	tp.HandleCancel(p.onFrameCancel)
	tp.HandleError(p.onFrameError)
	tp.HandleRequestN(p.onFrameRequestN)
	tp.HandlePayload(p.onFramePayload)
	tp.HandleKeepalive(p.onFrameKeepalive)

	if p.responder != nil {
		tp.HandleRequestResponse(p.onFrameRequestResponse)
		tp.HandleMetadataPush(p.respondMetadataPush)
		tp.HandleFNF(p.onFrameFNF)
		tp.HandleRequestStream(p.onFrameRequestStream)
		tp.HandleRequestChannel(p.onFrameRequestChannel)
	}

	p.cond.L.Lock()
	p.tp = tp
	p.cond.Signal()
	p.cond.L.Unlock()
}

func (p *DuplexRSocket) sendFrame(f framing.Frame) {
	defer func() {
		if e := recover(); e != nil {
			logger.Warnf("send frame failed: %s\n", e)
		}
	}()
	p.outs <- f
}

func (p *DuplexRSocket) sendPayload(
	sid uint32,
	sending payload.Payload,
	frameFlag framing.FrameFlag,
) {
	d := sending.Data()
	m, _ := sending.Metadata()
	size := framing.CalcPayloadFrameSize(d, m)

	if !p.shouldSplit(size) {
		p.sendFrame(framing.NewFramePayload(sid, d, m, frameFlag))
		return
	}
	p.doSplit(d, m, func(idx int, fg framing.FrameFlag, body *common.ByteBuff) {
		var h framing.FrameHeader
		if idx == 0 {
			h = framing.NewFrameHeader(sid, framing.FrameTypePayload, fg|frameFlag)
		} else {
			h = framing.NewFrameHeader(sid, framing.FrameTypePayload, fg|framing.FlagNext)
		}
		p.sendFrame(&framing.FramePayload{
			BaseFrame: framing.NewBaseFrame(h, body),
		})
	})
}

func (p *DuplexRSocket) drainWithKeepaliveAndLease(leaseChan <-chan lease.Lease) (ok bool) {
	if len(p.outs) > 0 {
		p.drain(nil)
	}
	var out framing.Frame
	select {
	case <-p.keepaliver.C():
		ok = true
		out = framing.NewFrameKeepalive(p.counter.ReadBytes(), nil, true)
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
		out = framing.NewFrameLease(ls.TimeToLive, ls.NumberOfRequests, ls.Metadata)
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

func (p *DuplexRSocket) drainWithKeepalive() (ok bool) {
	if len(p.outs) > 0 {
		p.drain(nil)
	}
	var out framing.Frame

	select {
	case <-p.keepaliver.C():
		ok = true
		out = framing.NewFrameKeepalive(p.counter.ReadBytes(), nil, true)
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

func (p *DuplexRSocket) drain(leaseChan <-chan lease.Lease) bool {
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
			if p.drainOne(framing.NewFrameLease(next.TimeToLive, next.NumberOfRequests, next.Metadata)) {
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

func (p *DuplexRSocket) drainOne(out framing.Frame) (wrote bool) {
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

func (p *DuplexRSocket) drainOutBack() {
	if len(p.outsPriority) < 1 {
		return
	}
	defer func() {
		p.outsPriority = p.outsPriority[:0]
	}()
	if p.tp == nil {
		return
	}
	var out framing.Frame
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

func (p *DuplexRSocket) loopWriteWithKeepaliver(ctx context.Context, leaseChan <-chan lease.Lease) error {
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
			kf := framing.NewFrameKeepalive(p.counter.ReadBytes(), nil, true)
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

func (p *DuplexRSocket) cleanOuts() {
	p.outsPriority = nil
}

func (p *DuplexRSocket) loopWrite(ctx context.Context) error {
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

func (p *DuplexRSocket) doSplit(data, metadata []byte, handler func(idx int, fg framing.FrameFlag, body *common.ByteBuff)) {
	fragmentation.Split(p.mtu, data, metadata, handler)
}

func (p *DuplexRSocket) doSplitSkip(skip int, data, metadata []byte, handler func(idx int, fg framing.FrameFlag, body *common.ByteBuff)) {
	fragmentation.SplitSkip(p.mtu, skip, data, metadata, handler)
}

func (p *DuplexRSocket) shouldSplit(size int) bool {
	return size > p.mtu
}

func (p *DuplexRSocket) register(sid uint32, msg interface{}) {
	p.messages.Store(sid, msg)
}

func (p *DuplexRSocket) unregister(sid uint32) {
	p.messages.Delete(sid)
	p.fragments.Delete(sid)
}

// NewServerDuplexRSocket creates a new server-side DuplexRSocket.
func NewServerDuplexRSocket(mtu int, leases lease.Leases) *DuplexRSocket {
	return &DuplexRSocket{
		closed:          atomic.NewBool(false),
		leases:          leases,
		outs:            make(chan framing.Frame, outsSize),
		mtu:             mtu,
		messages:        newU32Map(),
		sids:            &serverStreamIDs{},
		fragments:       newU32Map(),
		done:            make(chan struct{}),
		cond:            sync.NewCond(&sync.Mutex{}),
		counter:         transport.NewCounter(),
		singleScheduler: scheduler.NewSingle(64),
	}
}

// NewClientDuplexRSocket creates a new client-side DuplexRSocket.
func NewClientDuplexRSocket(
	mtu int,
	keepaliveInterval time.Duration,
) (s *DuplexRSocket) {
	ka := newKeepaliver(keepaliveInterval)
	s = &DuplexRSocket{
		closed:          atomic.NewBool(false),
		outs:            make(chan framing.Frame, outsSize),
		mtu:             mtu,
		messages:        newU32Map(),
		sids:            &clientStreamIDs{},
		fragments:       newU32Map(),
		done:            make(chan struct{}),
		cond:            sync.NewCond(&sync.Mutex{}),
		counter:         transport.NewCounter(),
		keepaliver:      ka,
		singleScheduler: scheduler.NewSingle(64),
	}
	return
}
