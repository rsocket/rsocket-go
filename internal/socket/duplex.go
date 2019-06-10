package socket

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/fragmentation"
	"github.com/rsocket/rsocket-go/internal/framing"
	"github.com/rsocket/rsocket-go/internal/logger"
	"github.com/rsocket/rsocket-go/internal/transport"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

const (
	outsSize  = 64
	outsSizeB = 16
)

var (
	errSocketClosed            = errors.New("socket closed already")
	unsupportedRequestStream   = []byte("Request-Stream not implemented.")
	unsupportedRequestResponse = []byte("Request-Response not implemented.")
	unsupportedRequestChannel  = []byte("Request-Channel not implemented.")
)

// DuplexRSocket represents a socket of RSocket which can be a requester or a responder.
type DuplexRSocket struct {
	counter      *transport.Counter
	tp           *transport.Transport
	outs         chan framing.Frame
	outsPriority chan framing.Frame
	responder    Responder
	messages     *mailboxes
	scheduler    rx.Scheduler
	sids         streamIDs
	mtu          int
	fragments    *sync.Map // key=streamID, value=Joiner
	once         *sync.Once
	done         chan struct{}
	keepaliver   *keepaliver
	cond         *sync.Cond
}

// Close close current socket.
func (p *DuplexRSocket) Close() (err error) {
	p.once.Do(func() {
		if p.keepaliver != nil {
			p.keepaliver.Stop()
		}
		close(p.outsPriority)
		close(p.outs)
		p.cond.L.Lock()
		p.cond.Broadcast()
		p.cond.L.Unlock()

		if p.tp != nil {
			err = p.tp.Close()
		}
		p.messages.each(func(k uint32, v *mailbox) {
			if v.receiving != nil {
				v.receiving.(interface{ Error(error) }).Error(errSocketClosed)
			}
			if v.sending != nil {
				v.sending.(interface{ Error(error) }).Error(errSocketClosed)
			}
		})
		p.fragments.Range(func(key, value interface{}) bool {
			value.(fragmentation.Joiner).Release()
			return true
		})
		<-p.done
	})
	return
}

// FireAndForget start a request of FireAndForget.
func (p *DuplexRSocket) FireAndForget(sending payload.Payload) {
	defer sending.Release()
	data := sending.Data()
	size := framing.HeaderLen + len(sending.Data())
	m, ok := sending.Metadata()
	if ok {
		size += 3 + len(m)
	}
	sid := p.sids.next()
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
	defer payload.Release()
	metadata, _ := payload.Metadata()
	p.sendFrame(framing.NewFrameMetadataPush(metadata))
}

// RequestResponse start a request of RequestResponse.
func (p *DuplexRSocket) RequestResponse(pl payload.Payload) rx.Mono {
	sid := p.sids.next()
	resp := rx.NewMono(nil)
	resp.DoAfterSuccess(func(ctx context.Context, elem payload.Payload) {
		elem.Release()
	})

	p.messages.put(sid, mailRequestResponse, nil, resp)

	data := pl.Data()
	metadata, _ := pl.Metadata()

	var stats int32

	merge := struct {
		sid   uint32
		d     []byte
		m     []byte
		r     rx.MonoProducer
		pl    payload.Payload
		stats *int32
	}{sid, data, metadata, resp.(rx.MonoProducer), pl, &stats}

	resp.
		DoFinally(func(ctx context.Context, sig rx.SignalType) {
			if sig == rx.SignalCancel {
				if atomic.LoadInt32(merge.stats) == 1 {
					p.sendFrame(framing.NewFrameCancel(merge.sid))
				} else {
					atomic.StoreInt32(merge.stats, -1)
				}
			}
			p.messages.remove(merge.sid)
		})

	// sending...
	defer func() {
		merge.pl.Release()
	}()
	// Check is it canceled?
	if atomic.LoadInt32(merge.stats) < 0 {
		return resp
	}
	size := framing.CalcPayloadFrameSize(data, metadata)
	if !p.shouldSplit(size) {
		p.sendFrame(framing.NewFrameRequestResponse(merge.sid, merge.d, merge.m))
		return resp
	}

	p.doSplit(merge.d, merge.m, func(idx int, fg framing.FrameFlag, body *common.ByteBuff) {
		var f framing.Frame
		if idx == 0 {
			h := framing.NewFrameHeader(merge.sid, framing.FrameTypeRequestResponse, fg)
			f = &framing.FrameRequestResponse{
				BaseFrame: framing.NewBaseFrame(h, body),
			}
		} else {
			h := framing.NewFrameHeader(merge.sid, framing.FrameTypePayload, fg|framing.FlagNext)
			f = &framing.FramePayload{
				BaseFrame: framing.NewBaseFrame(h, body),
			}
		}
		p.sendFrame(f)
	})
	return resp
}

// RequestStream start a request of RequestStream.
func (p *DuplexRSocket) RequestStream(sending payload.Payload) rx.Flux {
	sid := p.sids.next()
	flux := rx.NewFlux(nil)

	p.messages.put(sid, mailRequestStream, nil, flux)

	data := sending.Data()
	metadata, _ := sending.Metadata()

	// TODO: ugly code
	var sent int32
	merge := struct {
		sid  uint32
		d    []byte
		m    []byte
		l    payload.Payload
		sent *int32
	}{sid, data, metadata, sending, &sent}

	flux.
		DoAfterNext(func(ctx context.Context, elem payload.Payload) {
			elem.Release()
		}).
		DoFinally(func(ctx context.Context, sig rx.SignalType) {
			if sig == rx.SignalCancel {
				p.sendFrame(framing.NewFrameCancel(merge.sid))
			}
			p.messages.remove(merge.sid)

		}).
		DoOnRequest(func(ctx context.Context, n int) {
			if atomic.LoadInt32(merge.sent) > 0 {
				p.sendFrame(framing.NewFrameRequestN(merge.sid, uint32(n)))
			}
		}).
		DoOnSubscribe(func(ctx context.Context, s rx.Subscription) {
			defer func() {
				atomic.AddInt32(merge.sent, 1)
				merge.l.Release()
			}()
			initN := uint32(s.N())
			// reset initN if payload is RequestStream frame already.
			if rs, ok := merge.l.(*framing.FrameRequestStream); ok {
				initN = rs.InitialRequestN()
			}
			size := framing.CalcPayloadFrameSize(merge.d, merge.m) + 4
			if !p.shouldSplit(size) {
				p.sendFrame(framing.NewFrameRequestStream(merge.sid, initN, data, metadata))
				return
			}
			p.doSplitSkip(4, data, metadata, func(idx int, fg framing.FrameFlag, body *common.ByteBuff) {
				var f framing.Frame
				if idx == 0 {
					h := framing.NewFrameHeader(merge.sid, framing.FrameTypeRequestStream, fg)
					// write init RequestN
					binary.BigEndian.PutUint32(body.Bytes(), initN)
					f = &framing.FrameRequestStream{
						BaseFrame: framing.NewBaseFrame(h, body),
					}
				} else {
					h := framing.NewFrameHeader(merge.sid, framing.FrameTypePayload, fg|framing.FlagNext)
					f = &framing.FramePayload{
						BaseFrame: framing.NewBaseFrame(h, body),
					}
				}
				p.sendFrame(f)
			})
		})
	return flux
}

// RequestChannel start a request of RequestChannel.
func (p *DuplexRSocket) RequestChannel(publisher rx.Publisher) rx.Flux {
	sid := p.sids.next()
	sending := publisher.(rx.Flux)
	receiving := rx.NewFlux(nil)

	p.messages.put(sid, mailRequestChannel, sending, receiving)

	var idx uint32
	merge := struct {
		sid uint32
		i   *uint32
	}{sid, &idx}

	receiving.DoFinally(func(ctx context.Context, sig rx.SignalType) {
		// TODO: graceful close
		p.messages.remove(merge.sid)
	})

	sending.
		DoFinally(func(ctx context.Context, sig rx.SignalType) {
			// TODO: handle cancel or error
			switch sig {
			case rx.SignalComplete:
				p.sendFrame(framing.NewFramePayload(merge.sid, nil, nil, framing.FlagComplete))
			}
		}).
		DoOnSubscribe(func(ctx context.Context, s rx.Subscription) {
			s.Request(1)
		}).
		DoOnNext(func(ctx context.Context, s rx.Subscription, item payload.Payload) {
			if atomic.AddUint32(merge.i, 1) != 1 {
				p.sendPayload(merge.sid, item, true, framing.FlagNext)
				return
			}
			defer item.Release()

			// TODO: request N
			initN := uint32(rx.RequestInfinite)

			d := item.Data()
			m, _ := item.Metadata()
			size := framing.CalcPayloadFrameSize(d, m) + 4
			if !p.shouldSplit(size) {
				metadata, _ := item.Metadata()
				p.sendFrame(framing.NewFrameRequestChannel(merge.sid, initN, item.Data(), metadata, framing.FlagNext))
				return
			}
			p.doSplitSkip(4, d, m, func(idx int, fg framing.FrameFlag, body *common.ByteBuff) {
				var f framing.Frame
				if idx == 0 {
					h := framing.NewFrameHeader(merge.sid, framing.FrameTypeRequestChannel, fg|framing.FlagNext)
					// write init RequestN
					binary.BigEndian.PutUint32(body.Bytes(), initN)
					f = &framing.FrameRequestChannel{
						BaseFrame: framing.NewBaseFrame(h, body),
					}
				} else {
					h := framing.NewFrameHeader(merge.sid, framing.FrameTypePayload, fg|framing.FlagNext)
					f = &framing.FramePayload{
						BaseFrame: framing.NewBaseFrame(h, body),
					}
				}
				p.sendFrame(f)
			})
		}).
		SubscribeOn(p.scheduler).
		Subscribe(context.Background())

	return receiving
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
	sending, err := func() (mono rx.Mono, err error) {
		defer func() {
			err = toError(recover())
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
	// 4. register publisher
	p.messages.put(sid, mailRequestResponse, sending, nil)

	merge := struct {
		sid uint32
		r   fragmentation.HeaderAndPayload
	}{sid, receiving}

	// 5. async subscribe publisher
	sending.
		DoFinally(func(ctx context.Context, sig rx.SignalType) {
			p.messages.remove(merge.sid)
			merge.r.Release()
		}).
		DoOnError(func(ctx context.Context, err error) {
			p.writeError(merge.sid, err)
		}).
		SubscribeOn(p.scheduler).
		Subscribe(context.Background(), rx.OnNext(func(ctx context.Context, sub rx.Subscription, item payload.Payload) {
			p.sendPayload(merge.sid, item, true, framing.FlagNext|framing.FlagComplete)
		}))
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
		initRequestN = int(v.InitialRequestN())
	case fragmentation.Joiner:
		initRequestN = int(v.First().(*framing.FrameRequestChannel).InitialRequestN())
	default:
		panic("unreachable")
	}

	sid := pl.Header().StreamID()
	receiving := rx.NewFlux(nil)
	sending, err := func() (flux rx.Flux, err error) {
		defer func() {
			err = toError(recover())
		}()
		flux = p.responder.RequestChannel(receiving.(rx.Flux))
		if flux == nil {
			err = framing.NewFrameError(sid, common.ErrorCodeApplicationError, unsupportedRequestChannel)
		}
		return
	}()
	if err != nil {
		p.writeError(sid, err)
		return nil
	}

	p.messages.put(sid, mailRequestChannel, sending, receiving)

	if err := receiving.(rx.Producer).Next(pl); err != nil {
		pl.Release()
	}

	receiving.
		DoFinally(func(ctx context.Context, st rx.SignalType) {
			found, ok := p.messages.load(sid)
			if !ok {
				return
			}
			if found.sending == nil {
				p.messages.remove(sid)
			} else {
				found.receiving = nil
			}
		}).
		DoOnRequest(func(ctx context.Context, n int) {
			if n != math.MaxInt32 {
				p.sendFrame(framing.NewFrameRequestN(sid, uint32(n)))
			}
		}).
		DoOnSubscribe(func(ctx context.Context, s rx.Subscription) {
			n := uint32(s.N())
			if n == math.MaxInt32 {
				p.sendFrame(framing.NewFrameRequestN(sid, n))
			}
		})

	if receiving != sending {
		// auto release frame for each consumer
		receiving.DoAfterNext(func(ctx context.Context, item payload.Payload) {
			item.Release()
		})
	}

	sending.
		DoFinally(func(ctx context.Context, sig rx.SignalType) {
			found, ok := p.messages.load(sid)
			if !ok {
				return
			}
			if found.receiving == nil {
				p.messages.remove(sid)
			} else {
				found.sending = nil
			}
		}).
		DoOnError(func(ctx context.Context, err error) {
			p.writeError(sid, err)
		}).
		DoOnComplete(func(ctx context.Context) {
			p.sendFrame(framing.NewFramePayload(sid, nil, nil, framing.FlagComplete))
		}).
		SubscribeOn(p.scheduler).
		Subscribe(context.Background(), rx.OnSubscribe(func(ctx context.Context, s rx.Subscription) {
			s.Request(initRequestN)
		}), p.toSender(sid, framing.FlagNext))
	return nil
}

func (p *DuplexRSocket) respondMetadataPush(input framing.Frame) error {
	p.scheduler.Do(context.Background(), func(ctx context.Context) {
		defer input.Release()
		defer func() {
			if e := recover(); e != nil {
				logger.Errorf("respond metadata push failed: %s\n", e)
			}
		}()
		p.responder.MetadataPush(input.(*framing.FrameMetadataPush))
	})
	return nil
}

func (p *DuplexRSocket) onFrameFNF(frame framing.Frame) error {
	receiving, ok := p.doFragment(frame.(*framing.FrameFNF))
	if !ok {
		return nil
	}
	return p.respondFNF(receiving)
}

func (p *DuplexRSocket) respondFNF(receiving fragmentation.HeaderAndPayload) (err error) {
	p.scheduler.Do(context.Background(), func(ctx context.Context) {
		defer receiving.Release()
		defer func() {
			if e := recover(); e != nil {
				logger.Errorf("respond FireAndForget failed: %s\n", e)
			}
		}()
		p.responder.FireAndForget(receiving)
	})
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

	// 1. execute request stream handler
	resp, err := func() (resp rx.Flux, err error) {
		defer func() {
			err = toError(recover())
		}()
		resp = p.responder.RequestStream(receiving)
		if resp == nil {
			err = framing.NewFrameError(sid, common.ErrorCodeApplicationError, unsupportedRequestStream)
		}
		return
	}()

	// 2. send error with panic
	if err != nil {
		p.writeError(sid, err)
		return nil
	}

	// 3. register publisher
	p.messages.put(sid, mailRequestStream, resp, nil)

	// 4. seek initRequestN
	var initRequestN int
	switch v := receiving.(type) {
	case *framing.FrameRequestStream:
		initRequestN = int(v.InitialRequestN())
	case fragmentation.Joiner:
		initRequestN = int(v.First().(*framing.FrameRequestStream).InitialRequestN())
	default:
		panic("unreachable")
	}

	// 5. async subscribe publisher
	resp.
		DoFinally(func(ctx context.Context, sig rx.SignalType) {
			p.messages.remove(sid)
			receiving.Release()
		}).
		DoOnComplete(func(ctx context.Context) {
			p.sendFrame(framing.NewFramePayload(sid, nil, nil, framing.FlagComplete))
		}).
		DoOnError(func(ctx context.Context, err error) {
			p.writeError(sid, err)
		}).
		SubscribeOn(p.scheduler).
		Subscribe(context.Background(), rx.OnSubscribe(func(ctx context.Context, s rx.Subscription) {
			s.Request(initRequestN)
		}), p.toSender(sid, framing.FlagNext))
	return nil
}

func (p *DuplexRSocket) writeError(sid uint32, e error) {
	if v, ok := e.(*framing.FrameError); ok {
		p.sendFrame(v)
	} else {
		p.sendFrame(framing.NewFrameError(sid, common.ErrorCodeApplicationError, []byte(e.Error())))
	}
}

// SetResponder sets a responder for current socket.
func (p *DuplexRSocket) SetResponder(responder Responder) {
	p.responder = responder
}

func (p *DuplexRSocket) onFrameKeepalive(frame framing.Frame) (err error) {
	f := frame.(*framing.FrameKeepalive)
	if !f.Header().Flag().Check(framing.FlagRespond) {
		f.Release()
	} else {
		f.SetHeader(framing.NewFrameHeader(0, framing.FrameTypeKeepalive))
		p.sendFrame(f)
	}
	return
}

func (p *DuplexRSocket) onFrameCancel(frame framing.Frame) error {
	defer frame.Release()
	sid := frame.Header().StreamID()
	if v, ok := p.messages.load(sid); ok {
		v.sending.(rx.Disposable).Dispose()
	}
	if joiner, ok := p.fragments.Load(sid); ok {
		joiner.(fragmentation.Joiner).Release()
		p.fragments.Delete(sid)
	}
	return nil
}

func (p *DuplexRSocket) onFrameError(input framing.Frame) (err error) {
	f := input.(*framing.FrameError)
	logger.Errorf("handle error frame: %s\n", f)
	sid := f.Header().StreamID()

	v, ok := p.messages.load(sid)
	if !ok {
		return fmt.Errorf("invalid stream id: %d", sid)
	}
	switch v.mode {
	case mailRequestResponse:
		v.receiving.(rx.Mono).DoFinally(func(ctx context.Context, st rx.SignalType) {
			f.Release()
		})
		v.receiving.(rx.MonoProducer).Error(f)
	case mailRequestStream, mailRequestChannel:
		v.receiving.(rx.Flux).DoFinally(func(ctx context.Context, st rx.SignalType) {
			f.Release()
		})
		v.receiving.(rx.Producer).Error(f)
	default:
		panic("unreachable")
	}
	return
}

func (p *DuplexRSocket) onFrameRequestN(input framing.Frame) (err error) {
	defer input.Release()
	f := input.(*framing.FrameRequestN)
	sid := f.Header().StreamID()
	v, ok := p.messages.load(sid)
	if !ok {
		logger.Warnf("non-exists RequestN: id=%d", sid)
		return
	}
	// RequestN is always for sending.
	target := v.sending.(rx.Subscription)
	n := int(f.N())
	switch v.mode {
	case mailRequestStream, mailRequestChannel:
		target.(rx.Disposable).Dispose()
		target.Request(n)
	default:
		panic("unreachable")
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
	v, ok := p.messages.load(sid)
	if !ok {
		defer pl.Release()
		logger.Warnf("unoccupied Payload(id=%d), maybe it has been canceled", sid)
		return nil
	}
	fg := h.Flag()
	switch v.mode {
	case mailRequestResponse:
		if err := v.receiving.(rx.MonoProducer).Success(pl); err != nil {
			pl.Release()
			logger.Warnf("produce payload failed: %s\n", err.Error())
		}
	case mailRequestStream, mailRequestChannel:
		receiving := v.receiving.(rx.Producer)
		if fg.Check(framing.FlagNext) {
			if err := receiving.Next(pl); err != nil {
				pl.Release()
				logger.Warnf("produce payload failed: %s\n", err.Error())
			}
		}
		if fg.Check(framing.FlagComplete) {
			receiving.Complete()
			if !fg.Check(framing.FlagNext) {
				pl.Release()
			}
		}
	default:
		panic("unreachable")
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

func (p *DuplexRSocket) toSender(sid uint32, fg framing.FrameFlag) rx.OptSubscribe {
	merge := struct {
		tp  *transport.Transport
		sid uint32
		fg  framing.FrameFlag
	}{p.tp, sid, fg}
	return rx.OnNext(func(ctx context.Context, sub rx.Subscription, elem payload.Payload) {
		p.sendPayload(sid, elem, true, merge.fg)
	})
}

func (p *DuplexRSocket) sendFrame(f framing.Frame) {
	defer func() {
		if e := recover(); e != nil {
			f.Release()
			logger.Warnf("send frame failed: %s\n", e)
		}
	}()
	p.outs <- f
}

func (p *DuplexRSocket) sendPayload(
	sid uint32,
	sending payload.Payload,
	autoRelease bool,
	frameFlag framing.FrameFlag,
) {
	if autoRelease {
		defer sending.Release()
	}
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

func (p *DuplexRSocket) loopWrite(ctx context.Context) error {
	defer func() {
		close(p.done)
		if p.keepaliver != nil {
			p.keepaliver.Stop()
		}
	}()
	var finish uint8
	for finish < 2 {
		if finish < 1 && p.tp == nil {
			p.cond.L.Lock()
			p.cond.Wait()
			p.cond.L.Unlock()
		}

		select {
		case <-ctx.Done():
			// TODO: release all frames in chan
			return ctx.Err()
		default:
		}

		if p.keepaliver != nil {
			select {
			case <-p.keepaliver.C():
				kf := framing.NewFrameKeepalive(p.counter.ReadBytes(), nil, true)
				if p.tp != nil {
					err := p.tp.Send(kf)
					if err != nil {
						logger.Errorf("send keepalive frame failed: %s\n", err.Error())
					}
				}
				kf.Release()
			default:
			}
		}

		select {
		case out, ok := <-p.outsPriority:
			if !ok {
				finish++
				continue
			}
			if p.tp != nil {
				if err := p.tp.Send(out); err != nil {
					logger.Errorf("send frame failed: %s\n", err.Error())
				}
			}
			out.Release()
		default:
		}

		if p.keepaliver != nil {
			select {
			case <-p.keepaliver.C():
				kf := framing.NewFrameKeepalive(p.counter.ReadBytes(), nil, true)
				if p.tp != nil {
					err := p.tp.Send(kf)
					if err != nil {
						logger.Errorf("send keepalive frame failed: %s\n", err.Error())
					}
				}
				kf.Release()
			case out, ok := <-p.outs:
				if !ok {
					finish++
					continue
				}
				if p.tp == nil {
					p.outsPriority <- out
				} else if err := p.tp.Send(out); err != nil {
					p.outsPriority <- out
					logger.Errorf("send frame failed: %s\n", err.Error())
				} else {
					out.Release()
				}
			}
		} else {
			out, ok := <-p.outs
			if !ok {
				finish++
				continue
			}
			if p.tp == nil {
				p.outsPriority <- out
			} else if err := p.tp.Send(out); err != nil {
				p.outsPriority <- out
				logger.Errorf("send frame failed: %s\n", err.Error())
			} else {
				out.Release()
			}
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

// NewServerDuplexRSocket creates a new server-side DuplexRSocket.
func NewServerDuplexRSocket(mtu int, scheduler rx.Scheduler) *DuplexRSocket {
	return &DuplexRSocket{
		outs:         make(chan framing.Frame, outsSize),
		outsPriority: make(chan framing.Frame, outsSizeB),
		mtu:          mtu,
		messages:     newMailboxes(),
		scheduler:    scheduler,
		sids:         &serverStreamIDs{},
		fragments:    &sync.Map{},
		once:         &sync.Once{},
		done:         make(chan struct{}),
		cond:         sync.NewCond(&sync.Mutex{}),
		counter:      transport.NewCounter(),
	}
}

// NewClientDuplexRSocket creates a new client-side DuplexRSocket.
func NewClientDuplexRSocket(
	mtu int,
	scheduler rx.Scheduler,
	keepaliveInterval time.Duration,
) (s *DuplexRSocket) {
	ka := newKeepaliver(keepaliveInterval)
	s = &DuplexRSocket{
		outs:         make(chan framing.Frame, outsSize),
		outsPriority: make(chan framing.Frame, outsSizeB),
		mtu:          mtu,
		messages:     newMailboxes(),
		scheduler:    scheduler,
		sids:         &clientStreamIDs{},
		fragments:    &sync.Map{},
		once:         &sync.Once{},
		done:         make(chan struct{}),
		cond:         sync.NewCond(&sync.Mutex{}),
		counter:      transport.NewCounter(),
		keepaliver:   ka,
	}
	return
}
