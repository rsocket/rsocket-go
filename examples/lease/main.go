package main

import (
	"context"
	"log"
	"time"

	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/lease"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx/mono"
)

func main() {
	les, _ := lease.NewSimpleLease(10*time.Second, 7*time.Second, 1*time.Second, 5)
	err := rsocket.Receive().
		Lease(les).
		Acceptor(func(setup payload.SetupPayload, sendingSocket rsocket.CloseableRSocket) (socket rsocket.RSocket, e error) {
			socket = rsocket.NewAbstractSocket(
				rsocket.RequestResponse(func(msg payload.Payload) mono.Mono {
					return mono.Just(msg)
				}),
			)
			return
		}).
		Transport("tcp://127.0.0.1:7878").
		Serve(context.Background())

	if err != nil {
		log.Fatal(err)
	}
}
