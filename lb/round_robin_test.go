package lb_test

import (
	"context"
	"fmt"
	"log"
	"sync"
	"testing"

	. "github.com/rsocket/rsocket-go"
	. "github.com/rsocket/rsocket-go/lb"
	. "github.com/rsocket/rsocket-go/payload"
	. "github.com/rsocket/rsocket-go/rx"
	"github.com/stretchr/testify/require"
)

func TestRoundRobin(t *testing.T) {
	b := NewRoundRobin()
	defer b.Close()

	const x, y = 3, 10
	wg := &sync.WaitGroup{}
	wg.Add(x * y)
	for i := 0; i < x; i++ {
		go func(n int) {
			for j := 0; j < y; j++ {
				c, _ := b.Next()
				c.RequestResponse(NewString(fmt.Sprintf("GO_%04d_%04d", n, j), "go")).
					DoOnSuccess(func(ctx context.Context, s Subscription, elem Payload) {
						m, _ := elem.MetadataUTF8()
						log.Println("elem:", elem.DataUTF8(), m)
					}).
					DoFinally(func(ctx context.Context, st SignalType) {
						wg.Done()
					}).
					SubscribeOn(ElasticScheduler()).
					Subscribe(context.Background())
			}
		}(i)
	}
	for i := 0; i < 3; i++ {
		uri := fmt.Sprintf("tcp://127.0.0.1:%d", 8000+i)
		c, err := Connect().
			SetupPayload(NewString(uri, "hello")).
			Transport(uri).
			Start(context.Background())
		require.NoError(t, err)
		b.Push(c)
	}
	wg.Wait()
}
