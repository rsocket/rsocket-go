package rsocket

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"testing"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func TestBQueue_SetRate(t *testing.T) {
	qu := newQueue(16, 0, 2)
	qu.onRequestN = func(i int32) {
		log.Println("n:", i)
	}
	go func() {
		defer qu.Close()
		for i := 0; i < 100; i++ {
			//time.Sleep(10 * time.Millisecond)
			_ = qu.Add(NewPayloadString(fmt.Sprintf("foo@%d", i), "aa"))
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
	qu := newQueue(16, 2, 0)
	qu.onRequestN = func(i int32) {
		log.Println("onRequestN:", i)
	}

	go func(ctx context.Context) {
		defer close(done)
		qu.RequestN(2)
		n := 0
		for {
			v, ok := qu.Poll(ctx)
			if !ok {
				break
			}
			n++
			log.Println("next:", v)
			if n%2 == 0 {
				qu.RequestN(2)
			}
		}
	}(context.Background())
	go func() {
		for i := 0; i < 1000; i++ {
			if err := qu.Add(NewPayloadString(fmt.Sprintf("foo@%d", i), "aa")); err != nil {
				log.Println("add err:", err)
			}
		}
		_ = qu.Close()
	}()
	<-done
}
