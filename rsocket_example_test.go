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
		Acceptor(func(ctx context.Context, setup payload.SetupPayload, sendingSocket rsocket.CloseableRSocket) (rsocket.RSocket, error) {
			return rsocket.NewAbstractSocket(
				rsocket.RequestResponse(func(msg payload.Payload) mono.Mono {
					log.Println("incoming request:", msg)
					return mono.Just(payload.NewString("Pong", time.Now().String()))
				}),
			), nil
		}).
		Transport(rsocket.TCPServer().SetAddr(":7878").Build()).
		Serve(context.Background())
	if err != nil {
		panic(err)
	}

	// Connect to a server.
	cli, err := rsocket.Connect().
		SetupPayload(payload.NewString("Hello World", "From Golang")).
		Transport(rsocket.TCPClient().SetHostAndPort("127.0.0.1", 7878).Build()).
		Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer cli.Close()
	cli.RequestResponse(payload.NewString("Ping", time.Now().String())).
		DoOnSuccess(func(elem payload.Payload) error {
			log.Println("incoming response:", elem)
			return nil
		}).
		Subscribe(context.Background())
}

func ExampleReceive() {
	err := rsocket.Receive().
		Resume(rsocket.WithServerResumeSessionDuration(30 * time.Second)).
		Fragment(65535).
		Acceptor(func(ctx context.Context, setup payload.SetupPayload, sendingSocket rsocket.CloseableRSocket) (rsocket.RSocket, error) {
			// Handle close.
			sendingSocket.OnClose(func(err error) {
				log.Println("sending socket is closed")
			})

			// You can reject connection. For example, do some authorization.
			// return nil, errors.New("ACCESS_DENY")

			// Request to client.
			sendingSocket.RequestResponse(payload.NewString("Ping", time.Now().String())).
				DoOnSuccess(func(elem payload.Payload) error {
					log.Println("response of Ping from client:", elem)
					return nil
				}).
				SubscribeOn(scheduler.Parallel()).
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
				rsocket.RequestChannel(func(initialRequest payload.Payload, requests flux.Flux) flux.Flux {
					return requests
				}),
			), nil
		}).
		Transport(rsocket.TCPServer().SetHostAndPort("127.0.0.1", 7878).Build()).
		Serve(context.Background())
	panic(err)
}

func ExampleConnect() {
	cli, err := rsocket.Connect().
		Resume(). // Enable RESUME.
		Lease().  // Enable LEASE.
		Fragment(4096).
		SetupPayload(payload.NewString("Hello", "World")).
		Acceptor(func(ctx context.Context, socket rsocket.RSocket) rsocket.RSocket {
			return rsocket.NewAbstractSocket(
				rsocket.RequestResponse(func(msg payload.Payload) mono.Mono {
					return mono.Just(payload.NewString("Pong", time.Now().String()))
				}),
			)
		}).
		Transport(rsocket.TCPClient().SetAddr("127.0.0.1:7878").Build()).
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
		DoOnSuccess(func(elem payload.Payload) error {
			log.Println("response:", elem)
			return nil
		}).
		Subscribe(context.Background())
	var s rx.Subscription
	// RequestStream with backpressure. (one by one)
	cli.RequestStream(payload.NewString("This is a RequestStream message.", "")).
		DoOnNext(func(elem payload.Payload) error {
			log.Println("next element in stream:", elem)
			s.Request(1)
			return nil
		}).
		Subscribe(context.Background(), rx.OnSubscribe(func(ctx context.Context, s rx.Subscription) {
			s.Request(1)
		}))
	// Simple RequestChannel.
	initialPayload := payload.NewString("This is a RequestChannel initial message.", "")
	sendFlux := flux.Create(func(ctx context.Context, s flux.Sink) {
		for i := 0; i < 3; i++ {
			s.Next(payload.NewString(fmt.Sprintf("This is a RequestChannel message #%d.", i), ""))
		}
		s.Complete()
	})
	cli.RequestChannel(initialPayload, sendFlux).
		DoOnNext(func(elem payload.Payload) error {
			log.Println("next element in channel:", elem)
			return nil
		}).
		Subscribe(context.Background())
}
