package rx_test

import (
	"context"
	"testing"

	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

var testPayload payload.Payload

func init() {
	testPayload = payload.NewString("foo", "bar")
}

func Benchmark_Mono(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rx.
				NewMono(func(ctx context.Context, sink rx.MonoProducer) {
					_ = sink.Success(testPayload)
				}).
				DoOnSuccess(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
				}).
				Subscribe(context.Background())
		}
	})
}

func Benchmark_MonoJust(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rx.
				JustMono(testPayload).
				Subscribe(context.Background())
		}
	})
}

func Benchmark_Flux(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rx.
				NewFlux(func(ctx context.Context, producer rx.Producer) {
					_ = producer.Next(testPayload)
					producer.Complete()
				}).
				Subscribe(context.Background())
		}
	})
}
