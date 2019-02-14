package rsocket

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
)

type duplexRSocket struct {
	c               RConnection
	sun             bool
	requestStreamID uint32
	messages        *sync.Map // sid -> rx
}

func (p *duplexRSocket) Close() error {
	return p.c.Close()
}

func (p *duplexRSocket) FireAndForget(payload Payload) {
	_ = p.c.Send(createFNF(p.nextStreamID(), payload.Data(), payload.Metadata()))
}

func (p *duplexRSocket) MetadataPush(payload Payload) {
	_ = p.c.Send(createMetadataPush(payload.Metadata()))
}

func (p *duplexRSocket) RequestResponse(payload Payload) Mono {
	defer payload.Release()
	sid := p.nextStreamID()

	mono := NewMono(nil)
	mono.(*implMono).setAfterSub(func(ctx context.Context, item Payload) {
		item.Release()
	})
	p.messages.Store(sid, mono)
	mono.DoFinally(func(ctx context.Context) {
		p.messages.Delete(sid)
	})
	if err := p.c.Send(createRequestResponse(sid, payload.Data(), payload.Metadata())); err != nil {
		// TODO: process error
		panic(err)
	}
	return mono
}

func (p *duplexRSocket) RequestStream(payload Payload) Flux {
	defer payload.Release()
	sid := p.nextStreamID()
	flux := newImplFlux()
	p.messages.Store(sid, flux)
	flux.setAfterConsumer(func(ctx context.Context, item Payload) {
		item.Release()
	})
	flux.DoFinally(func(ctx context.Context) {
		p.messages.Delete(sid)
	})
	reqStream := createRequestStream(sid, 0xFFFFFFFF, payload.Data(), payload.Metadata())
	if err := p.c.Send(reqStream); err != nil {
		// TODO: wrapper error flux
		panic(err)
	}
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
	ctx := context.WithValue(context.Background(), ctxKeyRequestChannel, &ctxRequestChannel{
		sid:  sid,
		conn: p.c,
	})
	inputs.
		DoFinally(func(ctx context.Context) {
			v := ctx.Value(ctxKeyRequestChannel).(*ctxRequestChannel)
			_ = v.conn.Send(createPayloadFrame(v.sid, nil, nil, FlagComplete))
		}).
		SubscribeOn(ElasticScheduler()).
		Subscribe(ctx, func(ctx context.Context, item Payload) {
			defer item.Release()
			v := ctx.Value(ctxKeyRequestChannel).(*ctxRequestChannel)
			// TODO: process error, request N
			if atomic.AddUint32(&(v.processed), 1) == 1 {
				_ = v.conn.Send(createRequestChannel(v.sid, 0xFFFFFFFF, item.Data(), item.Metadata(), FlagNext))
			} else {
				_ = v.conn.Send(createPayloadFrame(v.sid, item.Data(), item.Metadata(), FlagNext))
			}
		})
	return fx
}

type ctxKey uint8

const (
	ctxKeyRequestChannel ctxKey = iota
)

type ctxRequestChannel struct {
	sid       uint32
	processed uint32
	conn      RConnection
}

func (p *duplexRSocket) nextStreamID() uint32 {
	if p.sun {
		// 2,4,6,8...
		return 2 * atomic.AddUint32(&p.requestStreamID, 1)
	}
	// 1,3,5,7
	return 2*(atomic.AddUint32(&p.requestStreamID, 1)-1) + 1
}

func (p *duplexRSocket) bindResponder(socket RSocket) {
	p.c.HandleRequestChannel(func(f *frameRequestChannel) (err error) {
		sid := f.header.StreamID()
		initialRequestN := f.InitialRequestN()
		payloads, loaded := p.messages.LoadOrStore(sid, newImplFlux())
		if loaded {
			panic(fmt.Errorf("duplicated request channel: streamID=%d", sid))
		}
		ElasticScheduler().Do(context.Background(), func(ctx context.Context) {
			_ = p.c.Send(createRequestN(sid, initialRequestN))
			payloads.(Emitter).Next(f)
		})
		outputs := socket.RequestChannel(payloads.(Flux))

		// TODO: ugly codes... optimize
		if payloads.(*implFlux) != outputs {
			// auto release frame for each consumer
			payloads.(*implFlux).setAfterConsumer(func(ctx context.Context, item Payload) {
				item.Release()
			})
		}
		outputs.
			DoFinally(func(ctx context.Context) {
				_ = p.c.Send(createPayloadFrame(sid, nil, nil, FlagComplete))
			}).
			SubscribeOn(ElasticScheduler()).
			Subscribe(context.Background(), p.consumeAsSend(sid, FlagNext))
		return
	})

	p.c.HandleRequestResponse(func(f *frameRequestResponse) (err error) {
		res := socket.RequestResponse(f)
		p.messages.Store(f.header.StreamID(), res)
		res.
			DoFinally(func(ctx context.Context) {
				p.messages.Delete(f.header.StreamID())
			}).
			SubscribeOn(ElasticScheduler()).
			Subscribe(context.Background(), func(ctx context.Context, item Payload) {
				if item == f {
					// hack and reuse frame, reduce frame copy
					fg := FlagNext | FlagComplete
					if len(f.Metadata()) > 0 {
						fg |= FlagMetadata
					}
					f.setHeader(createHeader(f.header.StreamID(), tPayload, fg))
					_ = p.c.Send(f)
				} else {
					defer f.Release()
					_ = p.c.Send(createPayloadFrame(f.header.StreamID(), item.Data(), item.Metadata(), FlagNext|FlagComplete))
				}
			})
		return
	})
	p.c.HandleMetadataPush(func(f *frameMetadataPush) (err error) {
		mono := JustMono(f)
		p.messages.Store(f.header.StreamID(), mono)
		mono.
			DoFinally(func(ctx context.Context) {
				p.messages.Delete(f.header.StreamID())
			}).
			DoFinally(func(ctx context.Context) {
				f.Release()
			}).
			SubscribeOn(ElasticScheduler()).
			Subscribe(context.Background(), func(ctx context.Context, item Payload) {
				socket.MetadataPush(item)
			})
		return
	})
	p.c.HandleFNF(func(f *frameFNF) (err error) {
		mono := JustMono(f)
		p.messages.Store(f.header.StreamID(), mono)
		mono.
			DoFinally(func(ctx context.Context) {
				p.messages.Delete(f.header.StreamID())
			}).
			DoFinally(func(ctx context.Context) {
				f.Release()
			}).
			SubscribeOn(ElasticScheduler()).
			Subscribe(context.Background(), func(ctx context.Context, item Payload) {
				socket.FireAndForget(item)
			})
		return
	})
	p.c.HandleRequestStream(func(f *frameRequestStream) (err error) {
		sid := f.header.StreamID()
		stream := socket.RequestStream(f)
		stream.
			DoFinally(func(ctx context.Context) {
				p.messages.Delete(sid)
			}).
			DoFinally(func(ctx context.Context) {
				f.Release()
			}).
			DoFinally(func(ctx context.Context) {
				_ = p.c.Send(createPayloadFrame(sid, nil, nil, FlagComplete))
			}).
			SubscribeOn(ElasticScheduler()).
			Subscribe(context.Background(), p.consumeAsSend(sid, FlagNext))
		return
	})
}

func (p *duplexRSocket) start() {
	p.c.HandleRequestN(func(f *frameRequestN) (err error) {
		// TODO:
		//log.Println("handle requestN:", f)
		return nil
	})
	p.c.HandlePayload(func(f *framePayload) (err error) {
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
			_ = p.c.Send(v)
		default:
			defer item.Release()
			_ = p.c.Send(createPayloadFrame(sid, item.Data(), item.Metadata(), fg))
		}
	}
}

func newDuplexRSocket(conn RConnection, sun bool) *duplexRSocket {
	sk := &duplexRSocket{
		c:        conn,
		sun:      sun,
		messages: &sync.Map{},
	}
	//tk := time.NewTicker(1 * time.Second)
	//go func() {
	//	for {
	//		select {
	//		case <-tk.C:
	//			var ccc int
	//			sk.messages.Range(func(key, value interface{}) bool {
	//				ccc++
	//				return true
	//			})
	//			log.Println("[leak trace] messages count:", ccc)
	//		}
	//	}
	//}()
	defer sk.start()
	return sk
}
