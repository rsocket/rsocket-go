package rx

import (
	"context"
	"math"
	"sync"
	"testing"
)

func BenchmarkRoutine(b *testing.B) {
	wg := &sync.WaitGroup{}
	wg.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func(ctx context.Context) {
			for i := 0; i < 100000; i++ {
				math.Sincos(math.Pi)
			}
			wg.Done()
		}(context.Background())
	}
	wg.Wait()
}

func BenchmarkAnts(b *testing.B) {
	wg := &sync.WaitGroup{}
	wg.Add(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ElasticScheduler().
			Do(context.Background(), func(ctx context.Context) {
				for i := 0; i < 100000; i++ {
					math.Sincos(math.Pi)
				}
				wg.Done()
			})
	}
	wg.Wait()
}
