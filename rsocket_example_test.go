package rsocket_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jjeffcaii/reactor-go/scheduler"
	. "github.com/rsocket/rsocket-go"
	. "github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/rsocket/rsocket-go/rx/mono"
)

func Example() {
	// Serve a server
	err := Receive().
		Acceptor(func(setup SetupPayload, sendingSocket CloseableRSocket) RSocket {
			return NewAbstractSocket(
				RequestResponse(func(msg Payload) mono.Mono {
					log.Println("incoming request:", msg)
					return mono.Just(NewString("Pong", time.Now().String()))
				}),
			)
		}).
		Transport("tcp://127.0.0.1:7878").
		Serve(context.Background())
	if err != nil {
		panic(err)
	}

	// Connect to a server.
	cli, err := Connect().
		SetupPayload(NewString("Hello World", "From Golang")).
		Transport("tcp://127.0.0.1:7878").
		Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = cli.Close()
	}()
	cli.RequestResponse(NewString("Ping", time.Now().String())).
		DoOnSuccess(func(elem Payload) {
			log.Println("incoming response:", elem)
		}).
		Subscribe(context.Background())
}

func ExampleReceive() {
	err := Receive().
		Resume(WithServerResumeSessionDuration(30 * time.Second)).
		Fragment(65535).
		Acceptor(func(setup SetupPayload, sendingSocket CloseableRSocket) RSocket {
			// Handle close.
			sendingSocket.OnClose(func() {
				log.Println("sending socket is closed")
			})
			// Request to client.
			sendingSocket.RequestResponse(NewString("Ping", time.Now().String())).
				DoOnSuccess(func(elem Payload) {
					log.Println("response of Ping from client:", elem)
				}).
				SubscribeOn(scheduler.Elastic()).
				Subscribe(context.Background())
			// Return responser which just echo.
			return NewAbstractSocket(
				FireAndForget(func(msg Payload) {
					log.Println("receive fnf:", msg)
				}),
				RequestResponse(func(msg Payload) mono.Mono {
					return mono.Just(msg)
				}),
				RequestStream(func(msg Payload) flux.Flux {
					return flux.Create(func(ctx context.Context, s flux.Sink) {
						for i := 0; i < 3; i++ {
							s.Next(NewString(msg.DataUTF8(), fmt.Sprintf("This is response #%04d", i)))
						}
						s.Complete()
					})
				}),
				RequestChannel(func(msgs rx.Publisher) flux.Flux {
					return msgs.(flux.Flux)
				}),
			)
		}).
		Transport("tcp://0.0.0.0:7878").
		Serve(context.Background())
	panic(err)
}

func ExampleConnect() {
	cli, err := Connect().
		Resume().
		Fragment(65535).
		SetupPayload(NewString("Hello", "World")).
		Acceptor(func(socket RSocket) RSocket {
			return NewAbstractSocket(RequestResponse(func(msg Payload) mono.Mono {
				return mono.Just(NewString("Pong", time.Now().String()))
			}))
		}).
		Transport("tcp://127.0.0.1:7878").
		Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = cli.Close()
	}()
	// Simple FireAndForget.
	cli.FireAndForget(NewString("This is a FNF message.", ""))
	// Simple RequestResponse.
	cli.RequestResponse(NewString("This is a RequestResponse message.", "")).
		DoOnSuccess(func(elem Payload) {
			log.Println("response:", elem)
		}).
		Subscribe(context.Background())
	var s rx.Subscription
	// RequestStream with backpressure. (one by one)
	cli.RequestStream(NewString("This is a RequestStream message.", "")).
		DoOnNext(func(elem Payload) {
			log.Println("next element in stream:", elem)
			s.Request(1)
		}).
		Subscribe(context.Background(), rx.OnSubscribe(func(s rx.Subscription) {
			s.Request(1)
		}))
	// Simple RequestChannel.
	sendFlux := flux.Create(func(ctx context.Context, s flux.Sink) {
		for i := 0; i < 3; i++ {
			s.Next(NewString(fmt.Sprintf("This is a RequestChannel message #%d.", i), ""))
		}
		s.Complete()
	})
	cli.RequestChannel(sendFlux).
		DoOnNext(func(elem Payload) {
			log.Println("next element in channel:", elem)
		}).
		Subscribe(context.Background())
}
