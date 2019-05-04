package rsocket

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/stretchr/testify/assert"
)

func TestLoadBalanceClient(t *testing.T) {
	setup := payload.NewString("hello", "world")

	discovery := make(chan []string)

	seeds := []string{
		"tcp://127.0.0.1:8000",
		"tcp://127.0.0.1:8001",
		"tcp://127.0.0.1:8002",
		"tcp://127.0.0.1:8003",
		"tcp://127.0.0.1:8004",
		"tcp://127.0.0.1:8005",
		"tcp://127.0.0.1:8006",
		"tcp://127.0.0.1:8007",
		"tcp://127.0.0.1:8008",
		"tcp://127.0.0.1:8009",
	}

	cli, err := Connect().SetupPayload(setup).
		Transports(discovery, WithInitTransports(seeds...)).
		Start(context.Background())
	if err != nil {
		assert.NoError(t, err, "cannot create client with load balance")
	}
	defer func() {
		_ = cli.Close()
	}()

	// simulate: random seeds every 5 seconds.
	tk := time.NewTicker(5 * time.Second)
	go func(ctx context.Context, tk *time.Ticker, discovery chan<- []string) {
		defer close(discovery)
		for {
			select {
			case <-ctx.Done():
				return
			case <-tk.C:
				foo := make(map[string]bool)
				n := len(seeds)
				for i := 0; i < n; i++ {
					foo[seeds[common.RandIntn(n)]] = true
				}
				seeds := make([]string, 0)
				for k := range foo {
					seeds = append(seeds, k)
				}
				discovery <- seeds
			}
		}
	}(context.Background(), tk, discovery)

	totals := 100000

	wg := &sync.WaitGroup{}
	wg.Add(totals)

	var success int32

	d := []byte(common.RandAlphanumeric(1024))
	m := []byte(common.RandAlphanumeric(1024))
	pl := payload.New(d, m)
	for i := 0; i < totals; i++ {
		cli.RequestResponse(pl).
			DoFinally(func(ctx context.Context, st rx.SignalType) {
				wg.Done()
			}).
			DoOnError(func(ctx context.Context, err error) {
				log.Println("oops:", err)
			}).
			DoOnSuccess(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
				//log.Println("rcv:", elem)
				atomic.AddInt32(&success, 1)
				m2, _ := elem.Metadata()
				assert.Equal(t, d, elem.Data(), "bad data")
				assert.Equal(t, m, m2, "bad metadata")
			}).
			SubscribeOn(rx.ElasticScheduler()).
			Subscribe(context.Background())
	}
	wg.Wait()
	log.Println("success:", success)
	assert.Equal(t, 0, common.CountByteBuffer(), "leak bytebuff")
}
