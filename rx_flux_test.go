package rsocket

import (
	"context"
	"fmt"
	"log"
	"testing"
)

func TestFluxProcessor_Request(t *testing.T) {
	f := NewFlux(func(ctx context.Context, producer Producer) {
		for i := 0; i < 1000; i++ {
			producer.Next(NewPayloadString(fmt.Sprintf("foo%d", i), fmt.Sprintf("bar%d", i)))
		}
		producer.Complete()
	})
	f.
		DoOnRequest(func(ctx context.Context, n int) {
			fmt.Println("n:", n)
		}).
		Subscribe(context.Background(), OnSubscribe(func(ctx context.Context, s Subscription) {
			s.Request(1)
		}), OnNext(func(ctx context.Context, sub Subscription, payload Payload) {
			fmt.Println(payload)
			sub.Request(1)
		}))
}

func TestFlux_Simple(t *testing.T) {
	f := NewFlux(func(ctx context.Context, producer Producer) {
		for i := 0; i < 3; i++ {
			producer.Next(NewPayloadString(fmt.Sprintf("foo%d", i), fmt.Sprintf("bar%d", i)))
		}
		producer.Complete()
	})
	f.
		DoFinally(func(ctx context.Context, st SignalType) {
			log.Println("doFinally:", st)
		}).
		DoOnNext(func(ctx context.Context, s Subscription, payload Payload) {
			log.Println("doNext:", payload)
		}).
		DoAfterNext(func(ctx context.Context, payload Payload) {
			payload.Release()
		}).
		Subscribe(context.Background())
}
