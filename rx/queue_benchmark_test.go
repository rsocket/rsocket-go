package rx

import (
	"context"
	"log"
	"testing"

	"github.com/rsocket/rsocket-go/common"
	"github.com/rsocket/rsocket-go/payload"
)

func BenchmarkQueue(b *testing.B) {
	qu := newQueue(16, 1, 0)

	pl := payload.NewString(common.RandAlphanumeric(1024), common.RandAlphanumeric(1024))

	ctx, cancel := context.WithCancel(context.Background())

	defer func() {
		cancel()
	}()

	go func(ctx context.Context) {
		defer func() {
			_ = qu.Close()
		}()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := qu.add(pl); err != nil {
					log.Println("add err:", err)
				}
			}
		}

	}(ctx)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		qu.requestN(1)
		_, ok := qu.poll(ctx)
		if !ok {
			break
		}
	}
}
