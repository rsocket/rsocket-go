package rsocket

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

var (
	unsupportedRequestStream   = []byte("Request-Stream not implemented.")
	unsupportedRequestResponse = []byte("Request-Response not implemented.")
	unsupportedRequestChannel  = []byte("Request-Channel not implemented.")
)

type duplexRSocket struct {
	responder       RSocket
	tp              transport
	serverMode      bool
	requestStreamID uint32
	messages        *sync.Map // sid -> rx
	scheduler       Scheduler
}

func (p *duplexRSocket) releaseAll() {
	disposables := make([]Disposable, 0)
	p.messages.Range(func(key, value interface{}) bool {
		disposables = append(disposables, value.(Disposable))
		return true
	})
	for _, value := range disposables {
		value.Dispose()
	}
}

func (p *duplexRSocket) Close() error {
	return p.tp.Close()
}

func (p *duplexRSocket) FireAndForget(payload Payload) {
	_ = p.tp.Send(createFNF(p.nextStreamID(), payload.Data(), payload.Metadata()))
}

func (p *duplexRSocket) MetadataPush(payload Payload) {
	_ = p.tp.Send(createMetadataPush(payload.Metadata()))
}

func (p *duplexRSocket) RequestResponse(payload Payload) Mono {
	sid := p.nextStreamID()
	resp := NewMono(nil)
	resp.DoAfterSuccess(func(ctx context.Context, item Payload) {
		item.Release()
	})
	p.setPublisher(sid, resp)
	resp.
		DoFinally(func(ctx context.Context, sig SignalType) {
			if sig == SignalCancel {
				_ = p.tp.Send(createCancel(sid))
			}
			p.unsetPublisher(sid)
		})

	p.scheduler.Do(context.Background(), func(ctx context.Context) {
		defer payload.Release()
		if err := p.tp.Send(createRequestResponse(sid, payload.Data(), payload.Metadata())); err != nil {
			resp.(MonoProducer).Error(err)
		}
	})
	return resp
}

func (p *duplexRSocket) RequestStream(payload Payload) Flux {
	sid := p.nextStreamID()
	flux := NewFlux(nil)
	p.setPublisher(sid, flux)

	merge := struct {
		sid uint32
		tp  transport
	}{sid, p.tp}

	flux.
		DoAfterNext(func(ctx context.Context, payload Payload) {
			payload.Release()
		}).
		DoFinally(func(ctx context.Context, sig SignalType) {
			p.unsetPublisher(sid)
			if sig == SignalCancel {
				_ = merge.tp.Send(createCancel(merge.sid))
			}
		}).
		DoOnRequest(func(ctx context.Context, n int) {
			_ = merge.tp.Send(createRequestN(merge.sid, uint32(n)))
		}).
		DoOnSubscribe(func(ctx context.Context, s Subscription) {
			defer payload.Release()
			if err := merge.tp.Send(createRequestStream(merge.sid, uint32(s.n()), payload.Data(), payload.Metadata())); err != nil {
				flux.(Producer).Error(err)
			}
		})
	return flux
}

func (p *duplexRSocket) RequestChannel(payloads Publisher) Flux {
	sid := p.nextStreamID()
	inputs := payloads.(Flux)
	fx := NewFlux(nil)
	p.messages.Store(sid, fx)
	fx.DoFinally(func(ctx context.Context, sig SignalType) {
		p.messages.Delete(sid)
	})

	var idx uint32
	merge := struct {
		tp  transport
		sid uint32
		i   *uint32
	}{p.tp, sid, &idx}

	inputs.
		DoFinally(func(ctx context.Context, sig SignalType) {
			_ = merge.tp.Send(createPayloadFrame(merge.sid, nil, nil, flagComplete))
		}).
		SubscribeOn(p.scheduler).
		Subscribe(context.Background(), OnNext(func(ctx context.Context, sub Subscription, item Payload) {
			defer item.Release()
			// TODO: request N
			if atomic.AddUint32(merge.i, 1) == 1 {
				_ = merge.tp.Send(createRequestChannel(merge.sid, math.MaxUint32, item.Data(), item.Metadata(), flagNext))
			} else {
				_ = merge.tp.Send(createPayloadFrame(merge.sid, item.Data(), item.Metadata(), flagNext))
			}
		}))

	return fx
}

func (p *duplexRSocket) respondRequestResponse(input Frame) error {
	// 0. do some convert jobs
	f := input.(*frameRequestResponse)
	sid := f.header.StreamID()
	// 1. execute socket handler
	send, err := func() (mono Mono, err error) {
		defer func() {
			err = toError(recover())
		}()
		mono = p.responder.RequestResponse(f)
		return
	}()
	// 2. send error with panic
	if err != nil {
		_ = p.writeError(sid, err)
		return nil
	}
	// 3. send error with unsupported handler
	if send == nil {
		_ = p.writeError(sid, createError(sid, ErrorCodeApplicationError, unsupportedRequestResponse))
		return nil
	}
	// 4. register publisher
	p.setPublisher(sid, send)
	// 5. async subscribe publisher
	send.
		DoFinally(func(ctx context.Context, sig SignalType) {
			p.unsetPublisher(sid)
			f.Release()
		}).
		DoOnError(func(ctx context.Context, err error) {
			_ = p.writeError(sid, err)
		}).
		SubscribeOn(p.scheduler).
		Subscribe(context.Background(), OnNext(func(ctx context.Context, sub Subscription, item Payload) {
			v, ok := item.(*frameRequestResponse)
			if !ok || v != f {
				_ = p.tp.Send(createPayloadFrame(sid, item.Data(), item.Metadata(), flagNext|flagComplete))
				return
			}
			// reuse request frame, reduce copy
			fg := flagNext | flagComplete
			if len(v.Metadata()) > 0 {
				fg |= flagMetadata
			}
			send := &framePayload{
				&baseFrame{
					header: createHeader(sid, tPayload, fg),
					body:   v.body,
				},
			}
			v.body = nil
			_ = p.tp.Send(send)
		}))
	return nil
}

func (p *duplexRSocket) respondRequestChannel(input Frame) error {
	f := input.(*frameRequestChannel)
	sid := f.header.StreamID()
	inputs := NewFlux(nil)
	outputs, err := func() (flux Flux, err error) {
		defer func() {
			err = toError(recover())
		}()
		flux = p.responder.RequestChannel(inputs.(Flux))
		if flux == nil {
			err = createError(sid, ErrorCodeApplicationError, unsupportedRequestChannel)
		}
		return
	}()
	if err != nil {
		return p.writeError(sid, err)
	}

	p.setPublisher(sid, inputs)
	initialRequestN := f.InitialRequestN()
	inputs.(Producer).Next(f)
	// TODO: process send error
	_ = p.tp.Send(createRequestN(sid, initialRequestN))

	if inputs != outputs {
		// auto release frame for each consumer
		inputs.DoAfterNext(func(ctx context.Context, item Payload) {
			item.Release()
		})
	}

	outputs.
		DoFinally(func(ctx context.Context, sig SignalType) {
			p.unsetPublisher(sid)
		}).
		DoOnError(func(ctx context.Context, err error) {
			_ = p.writeError(sid, err)
		}).
		DoOnComplete(func(ctx context.Context) {
			_ = p.tp.Send(createPayloadFrame(sid, nil, nil, flagComplete))
		}).
		SubscribeOn(p.scheduler).
		Subscribe(context.Background(), p.toSender(sid, flagNext))
	return nil
}

func (p *duplexRSocket) respondMetadataPush(input Frame) error {
	p.scheduler.Do(context.Background(), func(ctx context.Context) {
		defer input.Release()
		defer func() {
			if e := recover(); e != nil {
				logger.Errorf("respond metadata push failed: %s\n", e)
			}
		}()
		p.responder.MetadataPush(input.(*frameMetadataPush))
	})
	return nil
}

func (p *duplexRSocket) respondFNF(input Frame) error {
	p.scheduler.Do(context.Background(), func(ctx context.Context) {
		defer input.Release()
		defer func() {
			if e := recover(); e != nil {
				logger.Errorf("respond FireAndForget failed: %s\n", e)
			}
		}()
		p.responder.FireAndForget(input.(*frameFNF))
	})
	return nil
}

func (p *duplexRSocket) respondRequestStream(input Frame) error {
	f := input.(*frameRequestStream)
	sid := f.header.StreamID()

	// 1. execute request stream handler
	resp, err := func() (resp Flux, err error) {
		defer func() {
			err = toError(recover())
		}()
		resp = p.responder.RequestStream(f)
		if resp == nil {
			err = createError(sid, ErrorCodeApplicationError, unsupportedRequestStream)
		}
		return
	}()

	// 2. send error with panic
	if err != nil {
		return p.writeError(sid, err)
	}

	// 3. register publisher
	p.setPublisher(sid, resp)

	// 4. async subscribe publisher
	resp.
		DoFinally(func(ctx context.Context, sig SignalType) {
			f.Release()
			p.unsetPublisher(sid)
		}).
		DoOnComplete(func(ctx context.Context) {
			_ = p.tp.Send(createPayloadFrame(sid, nil, nil, flagComplete))
		}).
		DoOnError(func(ctx context.Context, err error) {
			_ = p.writeError(sid, err)
		}).
		SubscribeOn(p.scheduler).
		Subscribe(context.Background(), OnSubscribe(func(ctx context.Context, s Subscription) {
			s.Request(int(f.InitialRequestN()))
		}), p.toSender(sid, flagNext))
	return nil
}

func (p *duplexRSocket) writeError(sid uint32, err error) error {
	switch v := err.(type) {
	case *frameError:
		return p.tp.Send(v)
	case RError:
		return p.tp.Send(createError(sid, v.ErrorCode(), v.ErrorData()))
	default:
		return p.tp.Send(createError(sid, ErrorCodeApplicationError, []byte(err.Error())))
	}
}

func (p *duplexRSocket) bindResponder(socket RSocket) {
	p.responder = socket
	p.tp.handleRequestResponse(p.respondRequestResponse)
	p.tp.handleMetadataPush(p.respondMetadataPush)
	p.tp.handleFNF(p.respondFNF)
	p.tp.handleRequestStream(p.respondRequestStream)
	p.tp.handleRequestChannel(p.respondRequestChannel)
}

func (p *duplexRSocket) start() {
	p.tp.handleCancel(func(frame Frame) (err error) {
		logger.Warnf("incoming cancel frame: %s\n", frame)
		defer frame.Release()
		sid := frame.Header().StreamID()
		v, ok := p.messages.Load(sid)
		if !ok {
			return fmt.Errorf("invalid stream id: %d", sid)
		}
		v.(Disposable).Dispose()
		return nil
	})
	p.tp.handleError(func(input Frame) (err error) {
		f := input.(*frameError)
		logger.Errorf("handle error frame: %s\n", f)
		v, ok := p.messages.Load(f.header.StreamID())
		if !ok {
			return fmt.Errorf("invalid stream id: %d", f.header.StreamID())
		}
		switch foo := v.(type) {
		case Mono:
			foo.DoFinally(func(ctx context.Context, sig SignalType) {
				f.Release()
			})
		case Flux:
			foo.DoFinally(func(ctx context.Context, sig SignalType) {
				f.Release()
			})
		}
		switch foo := v.(type) {
		case MonoProducer:
			foo.Error(f)
		case Producer:
			foo.Error(f)
		}
		return
	})

	p.tp.handleRequestN(func(input Frame) (err error) {
		f := input.(*frameRequestN)
		defer f.Release()
		sid := f.header.StreamID()
		found, ok := p.messages.Load(sid)
		if !ok {
			return fmt.Errorf("non-exists stream id: %d", sid)
		}
		flux, ok := found.(Flux)
		if !ok {
			return fmt.Errorf("unsupport request n: streamId=%d", sid)
		}
		flux.(Subscription).Request(int(f.N()))
		return nil
	})

	p.tp.handlePayload(func(input Frame) (err error) {
		f := input.(*framePayload)
		sid := f.header.StreamID()
		v, ok := p.messages.Load(sid)
		if !ok {
			return fmt.Errorf("non-exist stream id: %d", sid)
		}
		switch vv := v.(type) {
		case MonoProducer:
			vv.Success(f)
		case Producer:
			if f.header.Flag().Check(flagNext) {
				vv.Next(f)
			} else if f.header.Flag().Check(flagComplete) {
				vv.Complete()
			}
		}
		return
	})
}

func (p *duplexRSocket) toSender(sid uint32, fg rFlags) OpSubscriber {
	merge := struct {
		tp  transport
		sid uint32
		fg  rFlags
	}{p.tp, sid, fg}
	return OnNext(func(ctx context.Context, sub Subscription, payload Payload) {
		switch v := payload.(type) {
		case *framePayload:
			if v.header.Flag().Check(flagMetadata) {
				v.setHeader(createHeader(merge.sid, tPayload, merge.fg|flagMetadata))
			} else {
				v.setHeader(createHeader(merge.sid, tPayload, merge.fg))
			}
			_ = merge.tp.Send(v)
		default:
			defer payload.Release()
			_ = merge.tp.Send(createPayloadFrame(merge.sid, payload.Data(), payload.Metadata(), merge.fg))
		}
	})
}

func (p *duplexRSocket) nextStreamID() uint32 {
	if p.serverMode {
		// 2,4,6,8...
		return 2 * atomic.AddUint32(&p.requestStreamID, 1)
	} else {
		// 1,3,5,7
		return 2*(atomic.AddUint32(&p.requestStreamID, 1)-1) + 1
	}
}

func (p *duplexRSocket) setPublisher(sid uint32, pub Publisher) {
	p.messages.Store(sid, pub)
}

func (p *duplexRSocket) unsetPublisher(sid uint32) {
	p.messages.Delete(sid)
}

func newDuplexRSocket(tp transport, serverMode bool, scheduler Scheduler) *duplexRSocket {
	sk := &duplexRSocket{
		tp:         tp,
		serverMode: serverMode,
		messages:   &sync.Map{},
		scheduler:  scheduler,
	}
	tp.onClose(func() {
		sk.releaseAll()
	})

	if logger.IsDebugEnabled() {
		done := make(chan struct{})
		tk := time.NewTicker(10 * time.Second)
		go func() {
			for {
				select {
				case <-done:
					return
				case <-tk.C:
					var ccc int
					sk.messages.Range(func(key, value interface{}) bool {
						ccc++
						return true
					})
					if ccc > 0 {
						logger.Debugf("[LEAK] messages count: %d\n", ccc)
					}
				}
			}
		}()
		tp.onClose(func() {
			tk.Stop()
			close(done)
		})
	}
	defer sk.start()
	return sk
}
