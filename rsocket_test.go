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

func ExampleReceive() {
	err := Receive().
		Resume().
		Fragment(1024).
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

func ExampleConnect() {
	cli, err := Connect().
		Resume().
		Fragment(1024).
		SetupPayload(NewString("Hello", "World")).
		Acceptor(func(socket RSocket) RSocket {
			return NewAbstractSocket(RequestResponse(func(msg Payload) Mono {
				return JustMono(NewString("Pong", time.Now().String()))
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
		DoOnSuccess(func(ctx context.Context, s Subscription, elem Payload) {
			log.Println("response:", elem)
		}).
		Subscribe(context.Background())
	// RequestStream with backpressure. (one by one)
	cli.RequestStream(NewString("This is a RequestStream message.", "")).
		DoOnNext(func(ctx context.Context, s Subscription, elem Payload) {
			log.Println("next element in stream:", elem)
			s.Request(1)
		}).
		DoOnSubscribe(func(ctx context.Context, s Subscription) {
			s.Request(1)
		}).
		Subscribe(context.Background())
	// Simple RequestChannel.
	sendFlux := Range(0, 3).
		Map(func(n int) Payload {
			return NewString(fmt.Sprintf("This is a RequestChannel message #%d.", n), "")
		})
	cli.RequestChannel(sendFlux).
		DoOnNext(func(ctx context.Context, s Subscription, elem Payload) {
			log.Println("next element in channel:", elem)
		}).
		Subscribe(context.Background())
}
