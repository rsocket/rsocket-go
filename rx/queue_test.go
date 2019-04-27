package rx

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/rsocket/rsocket-go/payload"
)

func TestBQueue_SetRate(t *testing.T) {
	qu := newQueue(16, 0, 2)
	qu.onRequestN = func(i int32) {
		log.Println("n:", i)
	}
	go func() {
		defer func() {
			_ = qu.Close()
		}()
		for i := 0; i < 100; i++ {
			//time.Sleep(10 * time.Millisecond)
			_ = qu.add(payload.NewString(fmt.Sprintf("foo@%d", i), "aa"))
		}
	}()
	for {
		v, ok := qu.poll(context.Background())
		if !ok {
			break
		}
		log.Println("onNext:", v)
	}
}

func TestQueue_Poll(t *testing.T) {
	done := make(chan struct{})
	qu := newQueue(16, 2, 0)
	qu.onRequestN = func(i int32) {
		log.Println("onRequestN:", i)
	}

	go func(ctx context.Context) {
		defer close(done)
		qu.requestN(2)
		n := 0
		for {
			v, ok := qu.poll(ctx)
			if !ok {
				break
			}
			n++
			log.Println("next:", v)
			if n%2 == 0 {
				qu.requestN(2)
			}
		}
	}(context.Background())
	go func() {
		for i := 0; i < 1000; i++ {
			if err := qu.add(payload.NewString(fmt.Sprintf("foo@%d", i), "aa")); err != nil {
				log.Println("add err:", err)
			}
		}
		_ = qu.Close()
	}()
	<-done
}
