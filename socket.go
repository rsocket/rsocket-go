package rsocket

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type duplexRSocket struct {
	tp              transport
	serverMode      bool
	requestStreamID uint32
	messages        *sync.Map // sid -> rx
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
	resp.(Mono).onAfterSubscribe(func(ctx context.Context, item Payload) {
		item.Release()
	})
	p.setPublisher(sid, resp)
	resp.
		DoFinally(func(ctx context.Context) {
			p.unsetPublisher(sid)
		}).
		DoOnCancel(func(ctx context.Context) {
			_ = p.tp.Send(createCancel(sid))
		})
	ElasticScheduler().Do(context.Background(), func(ctx context.Context) {
		defer payload.Release()
		if err := p.tp.Send(createRequestResponse(sid, payload.Data(), payload.Metadata())); err != nil {
			resp.(*implMono).Error(err)
		}
	})
	return resp
}

func (p *duplexRSocket) RequestStream(payload Payload) Flux {
	sid := p.nextStreamID()
	flux := newImplFlux()
	p.setPublisher(sid, flux)
	flux.
		onAfterSubscribe(func(ctx context.Context, item Payload) {
			item.Release()
		}).
		DoFinally(func(ctx context.Context) {
			p.unsetPublisher(sid)
		}).
		DoOnCancel(func(ctx context.Context) {
			_ = p.tp.Send(createCancel(sid))
		})

	ElasticScheduler().Do(context.Background(), func(ctx context.Context) {
		reqStream := createRequestStream(sid, 0xFFFFFFFF, payload.Data(), payload.Metadata())
		payload.Release()
		if err := p.tp.Send(reqStream); err != nil {
			reqStream.Release()
			flux.Error(err)
		}
	})
	return flux
}

func (p *duplexRSocket) RequestChannel(payloads Publisher) Flux {
	sid := p.nextStreamID()
	inputs := payloads.(Flux)
	fx := newImplFlux()
	p.messages.Store(sid, fx)
	fx.DoFinally(func(ctx context.Context) {
		p.messages.Delete(sid)
	})
	// TODO: ugly implements
	inputs.
		DoFinally(func(ctx context.Context) {
			_ = p.tp.Send(createPayloadFrame(sid, nil, nil, FlagComplete))
		}).
		SubscribeOn(ElasticScheduler()).
		subscribeIndexed(context.Background(), func(ctx context.Context, item Payload, i int) {
			defer item.Release()
			// TODO: process error, request N
			if i == 1 {
				_ = p.tp.Send(createRequestChannel(sid, 0xFFFFFFFF, item.Data(), item.Metadata(), FlagNext))
			} else {
				_ = p.tp.Send(createPayloadFrame(sid, item.Data(), item.Metadata(), FlagNext))
			}
		})
	return fx
}

func (p *duplexRSocket) respondRequestResponse(socket RSocket) frameHandler {
	return func(input Frame) (err error) {
		f := input.(*frameRequestResponse)
		sid := f.header.StreamID()
		send := socket.RequestResponse(f)
		if send == nil {
			send = NewMono(func(ctx context.Context, emitter MonoEmitter) {
				emitter.Error(createError(sid, ErrorCodeApplicationError, []byte("Request-Response not implemented.")))
			})
		}
		p.setPublisher(sid, send)
		send.
			DoFinally(func(ctx context.Context) {
				p.unsetPublisher(sid)
				f.Release()
			}).
			DoOnError(func(ctx context.Context, err error) {
				if v, ok := err.(*frameError); ok {
					_ = p.tp.Send(v)
				} else if v, ok := err.(RError); ok {
					_ = p.tp.Send(createError(sid, v.ErrorCode(), v.ErrorData()))
				} else {
					_ = p.tp.Send(createError(sid, ErrorCodeApplicationError, []byte(err.Error())))
				}
			}).
			SubscribeOn(ElasticScheduler()).
			Subscribe(context.Background(), func(ctx context.Context, item Payload) {
				if v, ok := item.(*frameRequestResponse); ok && v == f {
					fg := FlagNext | FlagComplete
					if len(v.Metadata()) > 0 {
						fg |= FlagMetadata
					}
					send := &framePayload{
						&baseFrame{
							header: createHeader(sid, tPayload, fg),
							body:   v.body,
						},
					}
					v.body = nil
					_ = p.tp.Send(send)
				} else {
					_ = p.tp.Send(createPayloadFrame(sid, item.Data(), item.Metadata(), FlagNext|FlagComplete))
				}
			})
		return
	}
}

func (p *duplexRSocket) respondRequestChannel(socket RSocket) frameHandler {
	return func(input Frame) (err error) {
		f := input.(*frameRequestChannel)
		sid := f.header.StreamID()
		initialRequestN := f.InitialRequestN()
		var inputs Flux = newImplFlux()
		p.setPublisher(sid, inputs)

		inputs.(Emitter).Next(f)
		// TODO: process send error
		_ = p.tp.Send(createRequestN(sid, initialRequestN))

		outputs := socket.RequestChannel(inputs.(Flux))
		if outputs == nil {
			outputs = NewFlux(func(ctx context.Context, emitter Emitter) {
				emitter.Error(createError(sid, ErrorCodeApplicationError, []byte("Request-Channel not implemented.")))
			})
		}

		if inputs != outputs {
			// auto release frame for each consumer
			inputs.onAfterSubscribe(func(ctx context.Context, item Payload) {
				item.Release()
			})
		}

		outputs.
			DoFinally(func(ctx context.Context) {
				p.unsetPublisher(sid)
			}).
			DoOnError(func(ctx context.Context, err error) {
				if v, ok := err.(*frameError); ok {
					_ = p.tp.Send(v)
				} else if v, ok := err.(RError); ok {
					_ = p.tp.Send(createError(sid, v.ErrorCode(), v.ErrorData()))
				} else {
					_ = p.tp.Send(createError(sid, ErrorCodeApplicationError, []byte(err.Error())))
				}
			}).
			DoOnComplete(func(ctx context.Context) {
				_ = p.tp.Send(createPayloadFrame(sid, nil, nil, FlagComplete))
			}).
			SubscribeOn(ElasticScheduler()).
			Subscribe(context.Background(), p.consumeAsSend(sid, FlagNext))
		return
	}
}

func (p *duplexRSocket) respondMetadataPush(socket RSocket) frameHandler {
	return func(input Frame) (err error) {
		f := input.(*frameMetadataPush)
		ElasticScheduler().Do(context.Background(), func(ctx context.Context) {
			defer f.Release()
			socket.MetadataPush(f)
		})
		return
	}
}

func (p *duplexRSocket) respondFNF(socket RSocket) frameHandler {
	return func(input Frame) (err error) {
		f := input.(*frameFNF)
		ElasticScheduler().Do(context.Background(), func(ctx context.Context) {
			defer f.Release()
			socket.FireAndForget(f)
		})
		return
	}
}

var (
	unsupportedRequestStream = []byte("Request-Stream not implemented.")
)

func (p *duplexRSocket) respondRequestStream(socket RSocket) frameHandler {
	return func(input Frame) (err error) {
		f := input.(*frameRequestStream)
		sid := f.header.StreamID()

		resp := socket.RequestStream(f)
		if resp == nil {
			resp = NewFlux(func(ctx context.Context, emitter Emitter) {
				emitter.Error(createError(sid, ErrorCodeApplicationError, unsupportedRequestStream))
			})
		}

		p.setPublisher(sid, resp)

		resp.
			DoFinally(func(ctx context.Context) {
				_ = p.tp.Send(createPayloadFrame(sid, nil, nil, FlagComplete))
				f.Release()
				p.unsetPublisher(sid)
			}).
			DoOnError(func(ctx context.Context, err error) {
				if v, ok := err.(*frameError); ok {
					_ = p.tp.Send(v)
				} else if v, ok := err.(RError); ok {
					_ = p.tp.Send(createError(sid, v.ErrorCode(), v.ErrorData()))
				} else {
					_ = p.tp.Send(createError(sid, ErrorCodeApplicationError, []byte(err.Error())))
				}
			}).
			SubscribeOn(ElasticScheduler()).
			Subscribe(context.Background(), p.consumeAsSend(sid, FlagNext))
		return
	}
}

func (p *duplexRSocket) bindResponder(socket RSocket) {
	p.tp.handleRequestResponse(p.respondRequestResponse(socket))
	p.tp.handleMetadataPush(p.respondMetadataPush(socket))
	p.tp.handleFNF(p.respondFNF(socket))
	p.tp.handleRequestStream(p.respondRequestStream(socket))
	p.tp.handleRequestChannel(p.respondRequestChannel(socket))
}

func (p *duplexRSocket) start() {
	p.tp.handleCancel(func(frame Frame) (err error) {
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
		logger.Errorf("error frame: %s\n", f)
		v, ok := p.messages.Load(f.header.StreamID())
		if !ok {
			return fmt.Errorf("invalid stream id: %d", f.header.StreamID())
		}
		foo := v.(*implMono)
		foo.DoFinally(func(ctx context.Context) {
			f.Release()
		})
		foo.Error(f)
		return
	})

	p.tp.handleRequestN(func(input Frame) (err error) {
		f := input.(*frameRequestN)
		// TODO: support flow control
		logger.Errorf("TODO: socket support incoming RequestN %s\n", f)
		return nil
	})
	p.tp.handlePayload(func(input Frame) (err error) {
		f := input.(*framePayload)
		v, ok := p.messages.Load(f.header.StreamID())
		if !ok {
			panic(fmt.Errorf("non-exist stream id: %d", f.header.StreamID()))
		}
		if mono, ok := v.(MonoEmitter); ok {
			ElasticScheduler().Do(context.Background(), func(ctx context.Context) {
				mono.Success(f)
			})
			return
		}
		flux := v.(Emitter)
		if f.header.Flag().Check(FlagNext) {
			flux.Next(f)
		} else if f.header.Flag().Check(FlagComplete) {
			flux.Complete()
		}
		return
	})
}

func (p *duplexRSocket) consumeAsSend(sid uint32, fg Flags) Consumer {
	return func(ctx context.Context, item Payload) {
		switch v := item.(type) {
		case *framePayload:
			if v.header.Flag().Check(FlagMetadata) {
				fg |= FlagMetadata
			}
			v.setHeader(createHeader(sid, tPayload, fg))
			_ = p.tp.Send(v)
		default:
			defer item.Release()
			_ = p.tp.Send(createPayloadFrame(sid, item.Data(), item.Metadata(), fg))
		}
	}
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

func newDuplexRSocket(tp transport, serverMode bool) *duplexRSocket {
	sk := &duplexRSocket{
		tp:         tp,
		serverMode: serverMode,
		messages:   &sync.Map{},
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
