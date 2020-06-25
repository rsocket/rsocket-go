package lease_test

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/lease"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx/mono"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

var _tp rsocket.Transporter

func init() {
	_tp = rsocket.Tcp().HostAndPort("127.0.0.1", 7979).Build()
}

func Init(ctx context.Context, started chan<- struct{}) {
	l, _ := lease.NewSimpleLease(10*time.Second, 7*time.Second, 1*time.Second, 5)
	err := rsocket.Receive().
		Lease(l).
		OnStart(func() {
			close(started)
		}).
		Acceptor(func(setup payload.SetupPayload, sendingSocket rsocket.CloseableRSocket) (rsocket.RSocket, error) {
			return rsocket.NewAbstractSocket(
				rsocket.RequestResponse(func(msg payload.Payload) mono.Mono {
					return mono.Just(msg)
				}),
			), nil
		}).
		Transport(_tp).
		Serve(ctx)
	if err != nil {
		log.Fatal(err)
	}
}

func TestClientWithLease(t *testing.T) {
	started := make(chan struct{})
	go Init(context.Background(), started)
	<-started

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cli, err := rsocket.Connect().
		Lease().
		Transport(_tp).
		Start(ctx)
	if err != nil {
		require.NoError(t, err, "connect failed")
	}
	defer cli.Close()

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
				fmt.Println("request failed:", err)
			} else {
				success.Inc()
				fmt.Println("request success:", v)
			}
		}
	}
	assert.Equal(t, uint32(10), success.Load(), "bad requests")
}
