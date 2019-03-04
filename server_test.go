package rsocket

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"
)

func TestXServer_Serve(t *testing.T) {
	err := Receive().
		Acceptor(func(setup SetupPayload, sendingSocket RSocket) RSocket {
			log.Println("setup:", string(setup.Data()))
			//duplexRSocket.FireAndForget(NewPayloadString("foo", "bar"))
			sendingSocket.RequestResponse(NewPayloadString("111", "222")).
				SubscribeOn(ElasticScheduler()).
				Subscribe(context.Background(), func(ctx context.Context, item Payload) {
					log.Println("FROM EARTH:", item)
				})

			return NewAbstractSocket(
				RequestResponse(func(payload Payload) Mono {
					return JustMono(payload)
				}),
				RequestStream(func(payload Payload) Flux {
					s := string(payload.Data())
					return NewFlux(func(ctx context.Context, emitter FluxEmitter) {
						for i := 0; i < 100; i++ {
							time.Sleep(100 * time.Millisecond)
							emitter.Next(NewPayload([]byte(fmt.Sprintf("%s_%d", s, i)), nil))
						}
						emitter.Complete()
					})
				}),
				FireAndForget(func(payload Payload) {
					log.Println("fireAndForget:", payload)
				}),
				RequestChannel(func(payloads Publisher) Flux {
					return payloads.(Flux)
				}),
			)
		}).
		Transport("127.0.0.1:8001").
		Serve()
	panic(err)
}
