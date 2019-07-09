package balancer_test

import (
	"context"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/jjeffcaii/reactor-go/scheduler"
	. "github.com/rsocket/rsocket-go"
	. "github.com/rsocket/rsocket-go/balancer"
	. "github.com/rsocket/rsocket-go/payload"
	. "github.com/rsocket/rsocket-go/rx"
)

func TestRoundRobin(t *testing.T) {
	b := NewRoundRobinBalancer()

	b.OnLeave(func(label string) {
		log.Println("client leave:", label)
	})
	defer func() {
		_ = b.Close()
	}()

	const x, y = 3, 1000
	wg := &sync.WaitGroup{}
	wg.Add(x * y)
	for i := 0; i < x; i++ {
		go func(n int) {
			for j := 0; j < y; j++ {
				b.Next().RequestResponse(NewString(fmt.Sprintf("GO_%04d_%04d", n, j), "go")).
					DoOnSuccess(func(elem Payload) {
						m, _ := elem.MetadataUTF8()
						log.Println("elem:", elem.DataUTF8(), m)
					}).
					DoFinally(func(st SignalType) {
						wg.Done()
					}).
					SubscribeOn(scheduler.Elastic()).
					Subscribe(context.Background())
				time.Sleep(1 * time.Second)
			}
		}(i)
	}
	for _, port := range []int{17878, 8000, 8001, 8002} {
		go func(uri string) {
			c, err := Connect().
				SetupPayload(NewString(uri, "hello")).
				Transport(uri).
				Start(context.Background())
			if err == nil {
				b.PutLabel(uri, c)
			}
		}(fmt.Sprintf("tcp://127.0.0.1:%d", port))
	}
	wg.Wait()
}
