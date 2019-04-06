package rsocket

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/stretchr/testify/assert"
)

func TestLoadBalanceClient(t *testing.T) {
	setup := payload.NewString("hello", "world")
	cli, err := Connect().SetupPayload(setup).
		Transport("tcp://127.0.0.1:8000", "tcp://127.0.0.1:7878").
		Start()
	if err != nil {
		assert.NoError(t, err, "cannot create client with load balance")
	}
	defer func() {
		_ = cli.Close()
	}()

	for i := 0; i < 1000; i++ {
		time.Sleep(100 * time.Millisecond)
		cli.RequestResponse(payload.NewString("hello", fmt.Sprintf("%d", i))).
			DoOnError(func(ctx context.Context, err error) {
				log.Println("oops:", err)
			}).
			DoOnSuccess(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
				log.Println("rcv:", elem)
			}).
			Subscribe(context.Background())
	}
}
