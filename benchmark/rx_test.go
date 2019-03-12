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
			NewMono(func(sink rsocket.MonoProducer) {
				sink.Success(payload)
			}).
			SubscribeOn(rsocket.ElasticScheduler()).
			Subscribe(context.Background(), rsocket.OnNext(func(ctx context.Context, sub rsocket.Subscription, payload rsocket.Payload) {
				wg.Done()
			}))
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
				Subscribe(context.Background(), rsocket.OnNext(func(ctx context.Context, sub rsocket.Subscription, payload rsocket.Payload) {
					wg.Done()
				}))
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
			NewFlux(func(ctx context.Context, producer rsocket.Producer) {
				producer.Next(payload)
				producer.Complete()
			}).
			DoFinally(func(ctx context.Context, sig rsocket.SignalType) {
				wg.Done()
			}).
			SubscribeOn(rsocket.ElasticScheduler()).
			Subscribe(ctx, rsocket.OnNext(func(ctx context.Context, sub rsocket.Subscription, payload rsocket.Payload) {

			}))
	}
	wg.Wait()
}
