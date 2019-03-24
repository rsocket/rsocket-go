package rsocket

import (
	"context"
	"errors"
	"fmt"
	"github.com/rsocket/rsocket-go/common"
	"github.com/rsocket/rsocket-go/common/logger"
	"github.com/rsocket/rsocket-go/framing"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/transport"
	"math"
	"sync"
	"sync/atomic"
)

var (
	unsupportedRequestStream   = []byte("Request-Stream not implemented.")
	unsupportedRequestResponse = []byte("Request-Response not implemented.")
	unsupportedRequestChannel  = []byte("Request-Channel not implemented.")
)

type duplexRSocket struct {
	responder       RSocket
	tp              transport.Transport
	serverMode      bool
	requestStreamID uint32
	messages        *publishersMap
	scheduler       rx.Scheduler
}

func (p *duplexRSocket) releaseAll() {
	p.messages.Dispose()
}

func (p *duplexRSocket) Close() error {
	return p.tp.Close()
}

func (p *duplexRSocket) FireAndForget(payload payload.Payload) {
	metadata, _ := payload.Metadata()
	_ = p.tp.Send(framing.NewFrameFNF(p.nextStreamID(), payload.Data(), metadata))
}

func (p *duplexRSocket) MetadataPush(payload payload.Payload) {
	metadata, _ := payload.Metadata()
	_ = p.tp.Send(framing.NewFrameMetadataPush(metadata))
}

func (p *duplexRSocket) RequestResponse(pl payload.Payload) rx.Mono {
	sid := p.nextStreamID()
	resp := rx.NewMono(nil)
	resp.DoAfterSuccess(func(ctx context.Context, elem payload.Payload) {
		elem.Release()
	})

	p.messages.put(sid, &publishers{
		mode:      msgStoreModeRequestResponse,
		receiving: resp,
	})

	resp.
		DoFinally(func(ctx context.Context, sig rx.SignalType) {
			if sig == rx.SignalCancel {
				_ = p.tp.Send(framing.NewFrameCancel(sid))
			}
			p.messages.remove(sid)
		})

	p.scheduler.Do(context.Background(), func(ctx context.Context) {
		defer pl.Release()
		metadata, _ := pl.Metadata()
		if err := p.tp.Send(framing.NewFrameRequestResponse(sid, pl.Data(), metadata)); err != nil {
			resp.(rx.MonoProducer).Error(err)
		}
	})
	return resp
}

func (p *duplexRSocket) RequestStream(elem payload.Payload) rx.Flux {
	sid := p.nextStreamID()
	flux := rx.NewFlux(nil)

	p.messages.put(sid, &publishers{
		mode:      msgStoreModeRequestStream,
		receiving: flux,
	})

	merge := struct {
		sid uint32
		tp  transport.Transport
	}{sid, p.tp}

	flux.
		DoAfterNext(func(ctx context.Context, elem payload.Payload) {
			elem.Release()
		}).
		DoFinally(func(ctx context.Context, sig rx.SignalType) {
			p.messages.remove(sid)
			if sig == rx.SignalCancel {
				_ = merge.tp.Send(framing.NewFrameCancel(merge.sid))
			}
		}).
		DoOnRequest(func(ctx context.Context, n int) {
			_ = merge.tp.Send(framing.NewFrameRequestN(merge.sid, uint32(n)))
		}).
		DoOnSubscribe(func(ctx context.Context, s rx.Subscription) {
			defer elem.Release()
			metadata, _ := elem.Metadata()
			if err := merge.tp.Send(framing.NewFrameRequestStream(merge.sid, uint32(s.N()), elem.Data(), metadata)); err != nil {
				flux.(rx.Producer).Error(err)
			}
		})
	return flux
}

func (p *duplexRSocket) RequestChannel(publisher rx.Publisher) rx.Flux {
	sid := p.nextStreamID()
	sending := publisher.(rx.Flux)
	receiving := rx.NewFlux(nil)

	p.messages.put(sid, &publishers{
		mode:      msgStoreModeRequestChannel,
		sending:   sending,
		receiving: receiving,
	})

	receiving.DoFinally(func(ctx context.Context, sig rx.SignalType) {
		// TODO: graceful close
		p.messages.remove(sid)
	})

	var idx uint32
	merge := struct {
		pubs *publishersMap
		tp   transport.Transport
		sid  uint32
		i    *uint32
	}{p.messages, p.tp, sid, &idx}

	sending.
		DoFinally(func(ctx context.Context, sig rx.SignalType) {
			if sig == rx.SignalComplete {
				_ = merge.tp.Send(framing.NewFramePayload(merge.sid, nil, nil, framing.FlagComplete))
			}
		}).
		SubscribeOn(p.scheduler).
		Subscribe(context.Background(), rx.OnNext(func(ctx context.Context, sub rx.Subscription, item payload.Payload) {
			defer item.Release()
			// TODO: request N
			if atomic.AddUint32(merge.i, 1) == 1 {
				metadata, _ := item.Metadata()
				_ = merge.tp.Send(framing.NewFrameRequestChannel(merge.sid, math.MaxUint32, item.Data(), metadata, framing.FlagNext))
			} else {
				metadata, _ := item.Metadata()
				_ = merge.tp.Send(framing.NewFramePayload(merge.sid, item.Data(), metadata, framing.FlagNext))
			}
		}))

	return receiving
}

func (p *duplexRSocket) respondRequestResponse(input framing.Frame) error {
	// 0. do some convert jobs
	receiving := input.(*framing.FrameRequestResponse)
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
		_ = p.writeError(sid, err)
		return nil
	}
	// 3. sending error with unsupported handler
	if sending == nil {
		_ = p.writeError(sid, framing.NewFrameError(sid, common.ErrorCodeApplicationError, unsupportedRequestResponse))
		return nil
	}
	// 4. register publisher
	p.messages.put(sid, &publishers{
		mode:    msgStoreModeRequestResponse,
		sending: sending,
	})
	// 5. async subscribe publisher
	sending.
		DoFinally(func(ctx context.Context, sig rx.SignalType) {
			p.messages.remove(sid)
			receiving.Release()
		}).
		DoOnError(func(ctx context.Context, err error) {
			_ = p.writeError(sid, err)
		}).
		SubscribeOn(p.scheduler).
		Subscribe(context.Background(), rx.OnNext(func(ctx context.Context, sub rx.Subscription, item payload.Payload) {
			v, ok := item.(*framing.FrameRequestResponse)
			if !ok || v != receiving {
				metadata, _ := item.Metadata()
				_ = p.tp.Send(framing.NewFramePayload(sid, item.Data(), metadata, framing.FlagNext|framing.FlagComplete))
				return
			}
			// reuse request frame, reduce copy
			fg := framing.FlagNext | framing.FlagComplete
			if _, ok := v.Metadata(); ok {
				fg |= framing.FlagMetadata
			}
			send := &framing.FramePayload{
				BaseFrame: framing.NewBaseFrame(framing.NewFrameHeader(sid, framing.FrameTypePayload, fg), v.Body()),
			}
			v.SetBody(nil)
			_ = p.tp.Send(send)
		}))
	return nil
}

func (p *duplexRSocket) respondRequestChannel(input framing.Frame) error {
	f := input.(*framing.FrameRequestChannel)
	sid := f.Header().StreamID()
	initN := int(f.InitialRequestN())
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
		return p.writeError(sid, err)
	}

	p.messages.put(sid, &publishers{
		mode:      msgStoreModeRequestChannel,
		sending:   sending,
		receiving: receiving,
	})

	if err := receiving.(rx.Producer).Next(f); err != nil {
		f.Release()
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
				_ = p.tp.Send(framing.NewFrameRequestN(sid, uint32(n)))
			}
		}).
		DoOnSubscribe(func(ctx context.Context, s rx.Subscription) {
			n := uint32(s.N())
			if n == math.MaxInt32 {
				_ = p.tp.Send(framing.NewFrameRequestN(sid, n))
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
			_ = p.writeError(sid, err)
		}).
		DoOnComplete(func(ctx context.Context) {
			_ = p.tp.Send(framing.NewFramePayload(sid, nil, nil, framing.FlagComplete))
		}).
		SubscribeOn(p.scheduler).
		Subscribe(context.Background(), rx.OnSubscribe(func(ctx context.Context, s rx.Subscription) {
			s.Request(initN)
		}), p.toSender(sid, framing.FlagNext))
	return nil
}

func (p *duplexRSocket) respondMetadataPush(input framing.Frame) error {
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

func (p *duplexRSocket) respondFNF(input framing.Frame) error {
	p.scheduler.Do(context.Background(), func(ctx context.Context) {
		defer input.Release()
		defer func() {
			if e := recover(); e != nil {
				logger.Errorf("respond FireAndForget failed: %s\n", e)
			}
		}()
		p.responder.FireAndForget(input.(*framing.FrameFNF))
	})
	return nil
}

func (p *duplexRSocket) respondRequestStream(input framing.Frame) error {
	f := input.(*framing.FrameRequestStream)
	sid := f.Header().StreamID()

	// 1. execute request stream handler
	resp, err := func() (resp rx.Flux, err error) {
		defer func() {
			err = toError(recover())
		}()
		resp = p.responder.RequestStream(f)
		if resp == nil {
			err = framing.NewFrameError(sid, common.ErrorCodeApplicationError, unsupportedRequestStream)
		}
		return
	}()

	// 2. send error with panic
	if err != nil {
		return p.writeError(sid, err)
	}

	// 3. register publisher
	p.messages.put(sid, &publishers{
		mode:    msgStoreModeRequestStream,
		sending: resp,
	})

	// 4. async subscribe publisher
	resp.
		DoFinally(func(ctx context.Context, sig rx.SignalType) {
			p.messages.remove(sid)
			f.Release()
		}).
		DoOnComplete(func(ctx context.Context) {
			_ = p.tp.Send(framing.NewFramePayload(sid, nil, nil, framing.FlagComplete))
		}).
		DoOnError(func(ctx context.Context, err error) {
			_ = p.writeError(sid, err)
		}).
		SubscribeOn(p.scheduler).
		Subscribe(context.Background(), rx.OnSubscribe(func(ctx context.Context, s rx.Subscription) {
			s.Request(int(f.InitialRequestN()))
		}), p.toSender(sid, framing.FlagNext))
	return nil
}

func (p *duplexRSocket) writeError(sid uint32, err error) error {
	v, ok := err.(*framing.FrameError)
	if ok {
		return p.tp.Send(v)
	}
	return p.tp.Send(framing.NewFrameError(sid, common.ErrorCodeApplicationError, []byte(err.Error())))
}

func (p *duplexRSocket) bindResponder(socket RSocket) {
	p.responder = socket
	p.tp.HandleRequestResponse(p.respondRequestResponse)
	p.tp.HandleMetadataPush(p.respondMetadataPush)
	p.tp.HandleFNF(p.respondFNF)
	p.tp.HandleRequestStream(p.respondRequestStream)
	p.tp.HandleRequestChannel(p.respondRequestChannel)
}

func (p *duplexRSocket) start() {
	p.tp.HandleCancel(func(frame framing.Frame) error {
		defer frame.Release()
		sid := frame.Header().StreamID()
		if v, ok := p.messages.load(sid); ok {
			v.sending.(rx.Disposable).Dispose()
		}
		return nil
	})
	p.tp.HandleError(func(input framing.Frame) (err error) {
		f := input.(*framing.FrameError)
		logger.Errorf("handle error frame: %s\n", f)
		sid := f.Header().StreamID()
		v, ok := p.messages.load(sid)
		if !ok {
			return fmt.Errorf("invalid stream id: %d", sid)
		}

		switch v.mode {
		case msgStoreModeRequestResponse:
			v.receiving.(rx.Mono).DoFinally(func(ctx context.Context, st rx.SignalType) {
				f.Release()
			})
			v.receiving.(rx.MonoProducer).Error(f)
		case msgStoreModeRequestStream, msgStoreModeRequestChannel:
			v.receiving.(rx.Flux).DoFinally(func(ctx context.Context, st rx.SignalType) {
				f.Release()
			})
			v.receiving.(rx.Producer).Error(f)
		default:
			panic("unreachable")
		}
		return
	})

	p.tp.HandleRequestN(func(input framing.Frame) (err error) {
		defer input.Release()
		f := input.(*framing.FrameRequestN)
		sid := f.Header().StreamID()
		v, ok := p.messages.load(sid)
		if !ok {
			return fmt.Errorf("non-exists stream id: %d", sid)
		}
		// RequestN is always for sending.
		target := v.sending.(rx.Subscription)
		n := int(f.N())
		switch v.mode {
		case msgStoreModeRequestStream, msgStoreModeRequestChannel:
			target.Request(n)
		default:
			panic("unreachable")
		}
		return
	})

	p.tp.HandlePayload(func(input framing.Frame) (err error) {
		f := input.(*framing.FramePayload)
		sid := f.Header().StreamID()
		v, ok := p.messages.load(sid)
		if !ok {
			return fmt.Errorf("non-exist stream id: %d", sid)
		}
		switch v.mode {
		case msgStoreModeRequestResponse:
			if err := v.receiving.(rx.MonoProducer).Success(f); err != nil {
				f.Release()
				logger.Warnf("produce payload failed: %s\n", err.Error())
			}
		case msgStoreModeRequestStream, msgStoreModeRequestChannel:
			receiving := v.receiving.(rx.Producer)
			fg := f.Header().Flag()
			if fg.Check(framing.FlagNext) {
				if err := receiving.Next(f); err != nil {
					f.Release()
					logger.Warnf("produce payload failed: %s\n", err.Error())
				}
			}
			if fg.Check(framing.FlagComplete) {
				receiving.Complete()
				if !fg.Check(framing.FlagNext) {
					f.Release()
				}
			}
		default:
			panic("unreachable")
		}
		return
	})
}

func (p *duplexRSocket) toSender(sid uint32, fg framing.FrameFlag) rx.OptSubscribe {
	merge := struct {
		tp  transport.Transport
		sid uint32
		fg  framing.FrameFlag
	}{p.tp, sid, fg}
	return rx.OnNext(func(ctx context.Context, sub rx.Subscription, elem payload.Payload) {
		switch v := elem.(type) {
		case *framing.FramePayload:
			if v.Header().Flag().Check(framing.FlagMetadata) {
				h := framing.NewFrameHeader(merge.sid, framing.FrameTypePayload, merge.fg|framing.FlagMetadata)
				v.SetHeader(h)
			} else {
				h := framing.NewFrameHeader(merge.sid, framing.FrameTypePayload, merge.fg)
				v.SetHeader(h)
			}
			_ = merge.tp.Send(v)
		default:
			defer elem.Release()
			metadata, _ := elem.Metadata()
			_ = merge.tp.Send(framing.NewFramePayload(merge.sid, elem.Data(), metadata, merge.fg))
		}
	})
}

func (p *duplexRSocket) nextStreamID() uint32 {
	if p.serverMode {
		// 2,4,6,8...
		return 2 * atomic.AddUint32(&p.requestStreamID, 1)
	}
	// 1,3,5,7
	return 2*(atomic.AddUint32(&p.requestStreamID, 1)-1) + 1
}

func newDuplexRSocket(tp transport.Transport, serverMode bool, scheduler rx.Scheduler) *duplexRSocket {
	sk := &duplexRSocket{
		tp:         tp,
		serverMode: serverMode,
		messages:   newMessageStore(),
		scheduler:  scheduler,
	}
	tp.OnClose(func() {
		sk.releaseAll()
	})
	sk.start()
	return sk
}

type msgStoreMode int8

const (
	msgStoreModeRequestResponse msgStoreMode = iota
	msgStoreModeRequestStream
	msgStoreModeRequestChannel
)

type publishers struct {
	mode               msgStoreMode
	sending, receiving rx.Publisher
}

type publishersMap struct {
	m *sync.Map
}

func (p *publishersMap) Dispose() {
	p.m.Range(func(key, value interface{}) bool {
		vv := value.(*publishers)
		if vv.receiving != nil {
			vv.receiving.(rx.Disposable).Dispose()
		}
		if vv.sending != nil {
			vv.sending.(rx.Disposable).Dispose()
		}
		return true
	})
}

func (p *publishersMap) put(id uint32, value *publishers) {
	p.m.Store(id, value)
}

func (p *publishersMap) load(id uint32) (v *publishers, ok bool) {
	found, ok := p.m.Load(id)
	if ok {
		v = found.(*publishers)
	}
	return
}

func (p *publishersMap) remove(id uint32) {
	p.m.Delete(id)
}

func newMessageStore() *publishersMap {
	return &publishersMap{
		m: &sync.Map{},
	}
}

// toError try convert something to error
func toError(err interface{}) error {
	if err == nil {
		return nil
	}
	switch v := err.(type) {
	case error:
		return v
	case string:
		return errors.New(v)
	default:
		return fmt.Errorf("%s", v)
	}
}
