package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strconv"

	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/rsocket/rsocket-go/rx/mono"
)

var tp transport.ServerTransporter

func init() {
	tp = rsocket.TCPServer().SetHostAndPort("127.0.0.1", 7878).Build()
	go func() {
		log.Println(http.ListenAndServe("127.0.0.1:4444", nil))
	}()
}

func main() {
	//logger.SetLevel(logger.LevelDebug)
	err := rsocket.Receive().
		OnStart(func() {
			log.Println("server start success!")
		}).
		Acceptor(func(ctx context.Context, setup payload.SetupPayload, sendingSocket rsocket.CloseableRSocket) (rsocket.RSocket, error) {
			//log.Println("SETUP BEGIN:----------------")
			//log.Println("maxLifeTime:", setup.MaxLifetime())
			//log.Println("keepaliveInterval:", setup.TimeBetweenKeepalive())
			//log.Println("dataMimeType:", setup.DataMimeType())
			//log.Println("metadataMimeType:", setup.MetadataMimeType())
			//log.Println("data:", string(setup.Data()))
			//log.Println("metadata:", string(setup.Metadata()))
			//log.Println("SETUP END:----------------")

			// NOTICE: request client for something.
			//sendingSocket.
			//	RequestResponse(payload.NewString("ping", "From server")).
			//	DoOnSuccess(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
			//		log.Println("rcv response from client:", elem)
			//	}).
			//	SubscribeOn(rx.ElasticScheduler()).
			//	Subscribe(context.Background())
			sendingSocket.OnClose(func(err error) {
				log.Println("*** socket disconnected ***", common.CountBorrowed())
			})
			// For SETUP_REJECT testing.
			//if strings.EqualFold(setup.DataUTF8(), "REJECT_ME") {
			//	return nil, errors.New("bye bye bye")
			//}
			return responder(), nil
		}).
		Transport(tp).
		Serve(context.Background())
	if err != nil {
		panic(err)
	}
}

func responder() rsocket.RSocket {
	return rsocket.NewAbstractSocket(
		rsocket.MetadataPush(func(item payload.Payload) {
			log.Println("GOT METADATA_PUSH:", item)
		}),
		rsocket.FireAndForget(func(elem payload.Payload) {
			log.Println("GOT FNF:", elem)
		}),
		rsocket.RequestResponse(func(pl payload.Payload) mono.Mono {
			// just echo
			return mono.JustOneshot(pl)

			// Graceful with context API.
			//return rx.NewMono(func(ctx context.Context, sink rx.MonoProducer) {
			//	time.Sleep(50 * time.Millisecond)
			//	select {
			//	case <-ctx.Done():
			//		break
			//	default:
			//		sink.Success(payload.Clone(pl))
			//	}
			//})
		}),
		rsocket.RequestStream(func(pl payload.Payload) flux.Flux {
			// for test: client metadata is totals as string

			// Here is my Java client code:
			//@Test
			//public void testRequestStream() {
			//	final int totals = 20;
			//	final CountDownLatch cdl = new CountDownLatch(totals);
			//	this.client.requestStream(
			//		DefaultPayload.create(RandomStringUtils.randomAlphanumeric(32), String.valueOf(totals)))
			//	.doOnNext(payload -> {
			//		log.info("income: data={}, metadata={}", payload.getDataUtf8(), payload.getMetadataUtf8());
			//		cdl.countDown();
			//	})
			//	.subscribe();
			//	try {
			//		cdl.await();
			//	} catch (InterruptedException e) {
			//		Thread.currentThread().interrupt();
			//		throw new Error(e);
			//	}
			//}
			s := pl.DataUTF8()
			m, _ := pl.MetadataUTF8()
			log.Println("data:", s, "metadata:", m)
			totals := 10
			if n, err := strconv.Atoi(m); err == nil {
				totals = n
			}
			return flux.Create(func(ctx context.Context, emitter flux.Sink) {
				for i := 0; i < totals; i++ {
					// You can use context for graceful coroutine shutdown, stop produce.
					select {
					case <-ctx.Done():
						log.Println("ctx done:", ctx.Err())
						return
					default:
						//time.Sleep(10 * time.Millisecond)
						emitter.Next(payload.NewString(fmt.Sprintf("%s_%d", s, i), m))
					}
				}
				emitter.Complete()
			})
		}),
		rsocket.RequestChannel(func(initialRequest payload.Payload, payloads flux.Flux) flux.Flux {
			//return payloads.(flux.Flux)
			payloads.
				//LimitRate(1).
				DoOnNext(func(next payload.Payload) error {
					log.Println("receiving:", next.DataUTF8())
					return nil
				}).
				Subscribe(context.Background())
			return flux.Create(func(i context.Context, sink flux.Sink) {
				for i := 0; i < 3; i++ {
					sink.Next(payload.NewString("world", fmt.Sprintf("%d", i)))
				}
				sink.Complete()
			})

			//return payloads.(rx.Flux)
			// echo all incoming payloads
			//return rx.NewFlux(func(ctx context.Context, emitter rx.Producer) {
			//	payloads.(rx.Flux).
			//		DoFinally(func(ctx context.Context, st rx.SignalType) {
			//			emitter.Complete()
			//		}).
			//		DoOnNext(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
			//			metadata, _ := elem.Metadata()
			//			_ = emitter.Next(payload.New(elem.Data(), metadata))
			//		}).
			//		SubscribeOn(rx.ElasticScheduler()).
			//		Subscribe(context.Background())
			//})
		}),
	)
}
