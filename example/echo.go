package main

import (
	"context"
	"fmt"
	"github.com/rsocket/rsocket-go"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strconv"
	"time"
)

func init() {
	go func() {
		log.Println(http.ListenAndServe(":4444", nil))
	}()
}

func main() {
	//rsocket.SetLoggerLevel(rsocket.LogLevelDebug)
	err := createEchoServer("127.0.0.1", 8001)
	panic(err)
}

func createEchoServer(host string, port int) error {
	responder := rsocket.NewAbstractSocket(
		rsocket.MetadataPush(func(payload rsocket.Payload) {
			log.Println("GOT METADATA_PUSH:", payload)
		}),
		rsocket.FireAndForget(func(payload rsocket.Payload) {
			log.Println("GOT FNF:", payload)
		}),
		rsocket.RequestResponse(func(payload rsocket.Payload) rsocket.Mono {
			// just echo
			return rsocket.JustMono(payload)
		}),
		rsocket.RequestStream(func(payload rsocket.Payload) rsocket.Flux {
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
			//		throw new Errorf(e);
			//	}
			//}
			s := string(payload.Data())
			m := string(payload.Metadata())
			log.Println("data:", s, "metadata:", m)
			totals := 100
			if n, err := strconv.Atoi(string(payload.Metadata())); err == nil {
				totals = n
			}
			return rsocket.NewFlux(func(ctx context.Context, emitter rsocket.FluxEmitter) {
				for i := 0; i < totals; i++ {
					// You can use context for graceful coroutine shutdown, stop produce.
					select {
					case <-ctx.Done():
						log.Println("ctx done:", ctx.Err())
						return
					default:
						time.Sleep(10 * time.Millisecond)
						payload := rsocket.NewPayloadString(fmt.Sprintf("%s_%d", s, i), m)
						emitter.Next(payload)
					}
				}
				emitter.Complete()
			})
		}),
		rsocket.RequestChannel(func(payloads rsocket.Publisher) rsocket.Flux {
			return payloads.(rsocket.Flux)
			//// echo all incoming payloads
			//f := rsocket.NewFlux(func(emitter rsocket.FluxEmitter) {
			//	req.
			//		DoFinally(func() {
			//			emitter.Complete()
			//		}).
			//		SubscribeOn(rsocket.ElasticScheduler()).
			//		Subscribe(func(item rsocket.Payload) {
			//			emitter.Next(rsocket.NewPayload(item.Data(), item.Metadata()))
			//		})
			//})
			//return f
		}),
	)
	return rsocket.Receive().
		Acceptor(func(setup rsocket.SetupPayload, sendingSocket rsocket.RSocket) rsocket.RSocket {
			//log.Println("SETUP BEGIN:----------------")
			//log.Println("maxLifeTime:", setup.MaxLifetime())
			//log.Println("keepaliveInterval:", setup.TimeBetweenKeepalive())
			//log.Println("dataMimeType:", setup.DataMimeType())
			//log.Println("metadataMimeType:", setup.MetadataMimeType())
			//log.Println("data:", string(setup.Data()))
			//log.Println("metadata:", string(setup.Metadata()))
			//log.Println("SETUP END:----------------")

			//sendingSocket.
			//	RequestResponse(rsocket.NewPayloadString("ping", "From server")).
			//	SubscribeOn(rsocket.ElasticScheduler()).
			//	Subscribe(context.Background(), func(ctx context.Context, item rsocket.Payload) {
			//		//log.Println("rcv response from client:", item)
			//	})

			return responder
		}).
		Transport(fmt.Sprintf("%s:%d", host, port)).
		Serve()
}
