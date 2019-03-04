package benchmark

import (
	"context"
	"github.com/rsocket/rsocket-go"
	"sync"
	"testing"
)

var payload rsocket.Payload

func init() {
	payload = rsocket.NewPayloadString("foo", "bar")
}

func Benchmark_Mono(b *testing.B) {
	wg := &sync.WaitGroup{}
	wg.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsocket.
			NewMono(func(ctx context.Context, emitter rsocket.MonoEmitter) {
				emitter.Success(payload)
			}).
			SubscribeOn(rsocket.ElasticScheduler()).
			Subscribe(context.Background(), func(ctx context.Context, item rsocket.Payload) {
				wg.Done()
			})
	}
	wg.Wait()
}

func Benchmark_MonoJust(b *testing.B) {
	wg := &sync.WaitGroup{}
	wg.Add(b.N)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rsocket.JustMono(payload).
				SubscribeOn(rsocket.ElasticScheduler()).
				Subscribe(context.Background(), func(ctx context.Context, item rsocket.Payload) {
					wg.Done()
				})
		}
	})
	wg.Wait()
}

func Benchmark_Flux(b *testing.B) {
	wg := &sync.WaitGroup{}
	wg.Add(b.N)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rsocket.
			NewFlux(func(ctx context.Context, emitter rsocket.FluxEmitter) {
				emitter.Next(payload)
				emitter.Complete()
			}).
			DoFinally(func(ctx context.Context, sig rsocket.SignalType) {
				wg.Done()
			}).
			SubscribeOn(rsocket.ElasticScheduler()).
			Subscribe(ctx, func(ctx context.Context, item rsocket.Payload) {
			})
	}
	wg.Wait()
}
