package rx

import (
	"context"
	"fmt"
	"github.com/rsocket/rsocket-go/payload"
	"log"
	"testing"
)

func TestToFlux(t *testing.T) {
	pub := JustMono(payload.NewString("hello", "world"))
	ToFlux(pub).
		DoOnNext(func(ctx context.Context, s Subscription, elem payload.Payload) {
			log.Println(elem)
		}).
		Subscribe(context.Background())
}

func TestToMono(t *testing.T) {
	flux := NewFlux(func(ctx context.Context, producer Producer) {
		var stop bool
		for i := 0; i < 10; i++ {
			if stop {
				break
			}
			select {
			case <-ctx.Done():
				stop = true
				break
			default:
				_ = producer.Next(payload.NewString("hello", fmt.Sprintf("%d", i)))
			}
		}
		producer.Complete()
	})

	ToMono(flux).
		DoOnSuccess(func(ctx context.Context, s Subscription, elem payload.Payload) {
			log.Println("success:", elem)
		}).
		Subscribe(context.Background())
}

func TestRange(t *testing.T) {
	Range(0, 10).
		Map(func(n int) payload.Payload {
			return payload.NewString("hello", fmt.Sprintf("%d", n))
		}).
		DoOnNext(func(ctx context.Context, s Subscription, elem payload.Payload) {
			log.Println(elem)
		}).
		Subscribe(context.Background())
}

func TestNewFluxFromArray(t *testing.T) {
	flux := NewFluxFromArray(payload.NewString("hello", "1"), payload.NewString("hello", "2"), payload.NewString("hello", "3"))
	flux.
		DoOnNext(func(ctx context.Context, s Subscription, elem payload.Payload) {
			log.Println(elem)
		}).
		Subscribe(context.Background())
}
