package rx

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/rsocket/rsocket-go/payload"
)

func TestFluxProcessor_Request(t *testing.T) {
	f := NewFlux(func(ctx context.Context, producer Producer) {
		for i := 0; i < 1000; i++ {
			producer.Next(payload.NewString(fmt.Sprintf("foo%d", i), fmt.Sprintf("bar%d", i)))
		}
		producer.Complete()
	})
	f.
		DoOnRequest(func(ctx context.Context, n int) {
			fmt.Println("n:", n)
		}).
		Subscribe(context.Background(), OnSubscribe(func(ctx context.Context, s Subscription) {
			s.Request(1)
		}), OnNext(func(ctx context.Context, sub Subscription, elem payload.Payload) {
			fmt.Println(elem)
			sub.Request(1)
		}))
}

func TestFlux_RequestN(t *testing.T) {
	f := Range(0, 100).Map(func(n int) payload.Payload {
		return payload.NewString(fmt.Sprintf("foo%d", n), fmt.Sprintf("bar%d", n))
	})

	f.
		LimitRate(2).
		DoOnRequest(func(ctx context.Context, n int) {
			log.Println("request:", n)
		}).
		DoOnNext(func(ctx context.Context, s Subscription, elem payload.Payload) {
			log.Println("next:", elem)
			//s.Request(1)
		}).
		//DoOnSubscribe(func(ctx context.Context, s Subscription) {
		//	s.Request(1)
		//}).
		Subscribe(context.Background())

}

func TestFlux_Simple(t *testing.T) {
	f := NewFlux(func(ctx context.Context, producer Producer) {
		for i := 0; i < 3; i++ {
			producer.Next(payload.NewString(fmt.Sprintf("foo%d", i), fmt.Sprintf("bar%d", i)))
		}
		producer.Complete()
	})
	f.
		DoFinally(func(ctx context.Context, st SignalType) {
			log.Println("doFinally:", st)
		}).
		DoOnNext(func(ctx context.Context, s Subscription, elem payload.Payload) {
			log.Println("doNext:", elem)
		}).
		DoAfterNext(func(ctx context.Context, elem payload.Payload) {
			elem.Release()
		}).
		Subscribe(context.Background())
}
