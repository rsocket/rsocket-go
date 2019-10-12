package main_test

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"

	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/payload"
)

func TestClientWithLease(t *testing.T) {
	ctx, _ := context.WithTimeout(context.Background(), 20*time.Second)
	cli, err := rsocket.Connect().
		Lease().
		Transport("tcp://127.0.0.1:7878").
		Start(ctx)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = cli.Close()
	}()

	success := atomic.NewUint32(0)

Loop:
	for {
		select {
		case <-ctx.Done():
			break Loop
		default:
			time.Sleep(1 * time.Second)
			v, err := cli.RequestResponse(payload.NewString("hello world", "go")).Block(context.Background())
			if err != nil {
				log.Println("request failed:", err)
			} else {
				success.Inc()
				log.Println("request success:", v)
			}
		}
	}
	assert.Equal(t, uint32(10), success.Load(), "bad requests")
}
