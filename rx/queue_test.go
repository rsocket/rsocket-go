package rx

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"testing"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/stretchr/testify/assert"
)

func TestQueue_Hunger(t *testing.T) {
	const totals = 3
	const initTickets = 0
	var produce int32
	var consume int32
	qu := newQueue(1, initTickets)
	qu.HandleRequest(func(i int32) {
		_ = qu.Push(payload.NewString(fmt.Sprintf("elem_%04d", produce), "ccc"))
		atomic.AddInt32(&produce, 1)
		log.Println("n:", i)
	})

	for i := 0; i < totals; i++ {
		v, ok := qu.Poll(context.Background())
		if !ok {
			break
		}
		log.Println("onNext:", v)
		consume++
	}
	assert.Equal(t, produce, consume)
}

func TestQueue_Poll(t *testing.T) {
	done := make(chan struct{})
	qu := newQueue(16, 2)
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
		for i := 0; i < 11; i++ {
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
	qu := newQueue(16, 1)

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
		_, ok := qu.Poll(ctx)
		if !ok {
			break
		}
		qu.Request(1)
	}
}
