package rsocket_test

import (
	"context"
	"fmt"
	"log"
	"time"

	. "github.com/rsocket/rsocket-go"
	. "github.com/rsocket/rsocket-go/payload"
	. "github.com/rsocket/rsocket-go/rx"
)

func ExampleServer_Serve() {
	err := Receive().
		Acceptor(func(setup SetupPayload, sendingSocket CloseableRSocket) RSocket {
			// Handle close.
			sendingSocket.OnClose(func() {
				log.Println("sending socket is closed")
			})
			// Request to client.
			sendingSocket.RequestResponse(NewString("Ping", time.Now().String())).
				SubscribeOn(ElasticScheduler()).
				Subscribe(context.Background())
			// Return responser which just echo.
			return NewAbstractSocket(
				FireAndForget(func(msg Payload) {
					log.Println("receive fnf:", msg)
				}),
				RequestResponse(func(msg Payload) Mono {
					return JustMono(msg)
				}),
				RequestStream(func(msg Payload) Flux {
					return Range(0, 3).Map(func(n int) Payload {
						return NewString(msg.DataUTF8(), fmt.Sprintf("This is respone #%04d", n))
					})
				}),
				RequestChannel(func(msgs Publisher) Flux {
					return ToFlux(msgs)
				}),
			)
		}).
		Transport("tcp://0.0.0.0:7878").
		Serve()
	panic(err)
}
