package main

import (
	"context"
	"fmt"

	"github.com/rsocket/rsocket-go/extension"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
)

func main() {
	// Here is an example which consume Payload one by one.
	f := flux.Create(func(ctx context.Context, s flux.Sink) {
		for i := 0; i < 5; i++ {
			s.Next(payload.NewString(fmt.Sprintf("Hello@%d", i), extension.TextPlain.String()))
		}
		s.Complete()
	})

	var su rx.Subscription
	f.
		DoOnRequest(func(n int) {
			fmt.Printf("requesting next %d element......\n", n)
		}).
		Subscribe(
			context.Background(),
			rx.OnSubscribe(func(s rx.Subscription) {
				// Init Request 1 element.
				su = s
				su.Request(1)
			}),
			rx.OnNext(func(elem payload.Payload) error {
				// Consume element, do something...
				fmt.Println("bingo:", elem)
				// Request for next one manually.
				su.Request(1)
				return nil
			}),
		)
}
