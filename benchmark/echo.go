package benchmark

import (
	"context"
	"fmt"
	"github.com/rsocket/rsocket-go"
	"log"
	"net/http"
	_ "net/http/pprof"
)

func runEchoServer(host string, port int) error {
	go func() {
		log.Println(http.ListenAndServe(":4444", nil))
	}()

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
			s := string(payload.Data())
			return rsocket.NewFlux(func(ctx context.Context, emitter rsocket.Emitter) {
				for i := 0; i < 100; i++ {
					payload := rsocket.NewPayloadString(fmt.Sprintf("%s_%d", s, i), "")
					emitter.Next(payload)
				}
				emitter.Complete()
			})
		}),
		rsocket.RequestChannel(func(payloads rsocket.Publisher) rsocket.Flux {
			return payloads.(rsocket.Flux)
			//// echo all incoming payloads
			//f := rsocket.NewFlux(func(emitter rsocket.Emitter) {
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
			log.Println("SETUP BEGIN:----------------")
			log.Println("maxLifeTime:", setup.MaxLifetime())
			log.Println("keepaliveInterval:", setup.TimeBetweenKeepalive())
			log.Println("dataMimeType:", setup.DataMimeType())
			log.Println("metadataMimeType:", setup.MetadataMimeType())
			log.Println("data:", string(setup.Data()))
			log.Println("metadata:", string(setup.Metadata()))
			log.Println("SETUP END:----------------")

			sendingSocket.
				RequestResponse(rsocket.NewPayloadString("ping", "From server")).
				SubscribeOn(rsocket.ElasticScheduler()).
				Subscribe(context.Background(), func(ctx context.Context, item rsocket.Payload) {
					log.Println("rcv response from client:", item)
				})

			return responder
		}).
		Transport(fmt.Sprintf("%s:%d", host, port)).
		Serve()
}
