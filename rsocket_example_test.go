package rsocket_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/rsocket/rsocket-go/rx/mono"
)

func Example() {
	// Serve a server
	err := rsocket.Receive().
		Resume(). // Enable RESUME
		//Lease().
		Acceptor(func(setup payload.SetupPayload, sendingSocket rsocket.CloseableRSocket) (rsocket.RSocket, error) {
			return rsocket.NewAbstractSocket(
				rsocket.RequestResponse(func(msg payload.Payload) mono.Mono {
					log.Println("incoming request:", msg)
					return mono.Just(payload.NewString("Pong", time.Now().String()))
				}),
			), nil
		}).
		Transport("tcp://127.0.0.1:7878").
		Serve(context.Background())
	if err != nil {
		panic(err)
	}

	// Connect to a server.
	cli, err := rsocket.Connect().
		SetupPayload(payload.NewString("Hello World", "From Golang")).
		Transport("tcp://127.0.0.1:7878").
		Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = cli.Close()
	}()
	cli.RequestResponse(payload.NewString("Ping", time.Now().String())).
		DoOnSuccess(func(elem payload.Payload) {
			log.Println("incoming response:", elem)
		}).
		Subscribe(context.Background())
}

func ExampleReceive() {
	err := rsocket.Receive().
		Resume(rsocket.WithServerResumeSessionDuration(30 * time.Second)).
		Fragment(65535).
		Acceptor(func(setup payload.SetupPayload, sendingSocket rsocket.CloseableRSocket) (rsocket.RSocket, error) {
			// Handle close.
			sendingSocket.OnClose(func(err error) {
				log.Println("sending socket is closed")
			})

			// You can reject connection. For example, do some authorization.
			// return nil, errors.New("ACCESS_DENY")

			// Request to client.
			sendingSocket.RequestResponse(payload.NewString("Ping", time.Now().String())).
				DoOnSuccess(func(elem payload.Payload) {
					log.Println("response of Ping from client:", elem)
				}).
				SubscribeOn(scheduler.Elastic()).
				Subscribe(context.Background())
			// Return responser which just echo.
			return rsocket.NewAbstractSocket(
				rsocket.FireAndForget(func(msg payload.Payload) {
					log.Println("receive fnf:", msg)
				}),
				rsocket.RequestResponse(func(msg payload.Payload) mono.Mono {
					return mono.Just(msg)
				}),
				rsocket.RequestStream(func(msg payload.Payload) flux.Flux {
					return flux.Create(func(ctx context.Context, s flux.Sink) {
						for i := 0; i < 3; i++ {
							s.Next(payload.NewString(msg.DataUTF8(), fmt.Sprintf("This is response #%04d", i)))
						}
						s.Complete()
					})
				}),
				rsocket.RequestChannel(func(msgs rx.Publisher) flux.Flux {
					return msgs.(flux.Flux)
				}),
			), nil
		}).
		Transport("tcp://0.0.0.0:7878").
		Serve(context.Background())
	panic(err)
}

func ExampleConnect() {
	cli, err := rsocket.Connect().
		Resume(). // Enable RESUME.
		Lease().  // Enable LEASE.
		Fragment(4096).
		SetupPayload(payload.NewString("Hello", "World")).
		Acceptor(func(socket rsocket.RSocket) rsocket.RSocket {
			return rsocket.NewAbstractSocket(
				rsocket.RequestResponse(func(msg payload.Payload) mono.Mono {
					return mono.Just(payload.NewString("Pong", time.Now().String()))
				}),
			)
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
	cli.FireAndForget(payload.NewString("This is a FNF message.", ""))
	// Simple RequestResponse.
	cli.RequestResponse(payload.NewString("This is a RequestResponse message.", "")).
		DoOnSuccess(func(elem payload.Payload) {
			log.Println("response:", elem)
		}).
		Subscribe(context.Background())
	var s rx.Subscription
	// RequestStream with backpressure. (one by one)
	cli.RequestStream(payload.NewString("This is a RequestStream message.", "")).
		DoOnNext(func(elem payload.Payload) {
			log.Println("next element in stream:", elem)
			s.Request(1)
		}).
		Subscribe(context.Background(), rx.OnSubscribe(func(s rx.Subscription) {
			s.Request(1)
		}))
	// Simple RequestChannel.
	sendFlux := flux.Create(func(ctx context.Context, s flux.Sink) {
		for i := 0; i < 3; i++ {
			s.Next(payload.NewString(fmt.Sprintf("This is a RequestChannel message #%d.", i), ""))
		}
		s.Complete()
	})
	cli.RequestChannel(sendFlux).
		DoOnNext(func(elem payload.Payload) {
			log.Println("next element in channel:", elem)
		}).
		Subscribe(context.Background())
}
