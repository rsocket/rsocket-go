package main

import (
	"context"
	"fmt"

	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/logger"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
)

var testData = "Hello World!"

func main() {
	logger.SetLevel(logger.LevelDebug)
	err := rsocket.Receive().
		Fragment(128).
		Acceptor(func(setup payload.SetupPayload, sendingSocket rsocket.CloseableRSocket) rsocket.RSocket {
			return rsocket.NewAbstractSocket(
				rsocket.RequestChannel(func(inputs rx.Publisher) flux.Flux {
					//var count int32
					//countPointer := &count
					receives := make(chan payload.Payload)

					go func() {
						var count int32
						for range receives {
							count++
						}
						fmt.Println("***** count:", count)
					}()

					inputs.(flux.Flux).DoFinally(func(s rx.SignalType) {
						close(receives)
					}).Subscribe(context.Background(), rx.OnNext(func(input payload.Payload) {
						//fmt.Println("rcv from channel:", input)
						receives <- input
					}))

					return flux.Create(func(ctx context.Context, s flux.Sink) {
						for i := 0; i < 2; i++ {
							s.Next(payload.NewString(testData, fmt.Sprintf("%d_from_server", i)))
						}
						s.Complete()
					})
				}),
			)
		}).
		Transport("tcp://127.0.0.1:7878").
		Serve(context.Background())
	fmt.Println("SERVER STOPPED!!!!!")
	panic(err)
}
