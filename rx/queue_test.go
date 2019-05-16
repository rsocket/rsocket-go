package rx

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/rsocket/rsocket-go/common"
	"github.com/rsocket/rsocket-go/payload"
)

func TestBQueue_SetRate(t *testing.T) {
	qu := newQueue(16, 0, 2)
	qu.HandleRequest(func(i int32) {
		log.Println("n:", i)
	})
	go func() {
		defer func() {
			_ = qu.Close()
		}()
		for i := 0; i < 100; i++ {
			//time.Sleep(10 * time.Millisecond)
			_ = qu.Push(payload.NewString(fmt.Sprintf("foo@%d", i), "aa"))
		}
	}()
	for {
		v, ok := qu.Poll(context.Background())
		if !ok {
			break
		}
		log.Println("onNext:", v)
	}
}

func TestQueue_Poll(t *testing.T) {
	done := make(chan struct{})
	//qu := newQueue(16, 2, 0)
	qu := newQueue(16, 2, 0)
	qu.HandleRequest(func(i int32) {
		log.Println("onRequestN:", i)
	})

	go func(ctx context.Context) {
		defer close(done)
		qu.Request(2)
		n := 0
		for {
			v, ok := qu.Poll(ctx)
			if !ok {
				break
			}
			n++
			log.Println("next:", v)
			if n%2 == 0 {
				qu.Request(2)
			}
		}
	}(context.Background())
	go func() {
		for i := 0; i < 1000; i++ {
			if err := qu.Push(payload.NewString(fmt.Sprintf("foo@%d", i), "aa")); err != nil {
				log.Println("add err:", err)
			}
		}
		_ = qu.Close()
	}()
	<-done
}

func BenchmarkQueue(b *testing.B) {
	//qu := newQueue(16, 1, 0)
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
				if err := qu.Push(pl); err != nil {
					log.Println("add err:", err)
				}
			}
		}

	}(ctx)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		qu.Request(1)
		_, ok := qu.Poll(ctx)
		if !ok {
			break
		}
	}
}
