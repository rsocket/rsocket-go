package balancer_test

import (
	"context"
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/rsocket/rsocket-go"
	. "github.com/rsocket/rsocket-go/balancer"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/mono"
	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

func startServer(ctx context.Context, port int, counter *sync.Map) {
	_ = rsocket.Receive().
		Acceptor(func(setup payload.SetupPayload, sendingSocket rsocket.CloseableRSocket) (rsocket.RSocket, error) {
			return rsocket.NewAbstractSocket(
				rsocket.RequestResponse(func(msg payload.Payload) mono.Mono {
					cur, _ := counter.LoadOrStore(port, atomic.NewInt32(0))
					cur.(*atomic.Int32).Inc()
					return mono.Just(msg)
				}),
			), nil
		}).
		Transport(rsocket.TCPServer().SetHostAndPort("127.0.0.1", port).Build()).
		Serve(ctx)
}

func TestRoundRobin(t *testing.T) {
	ports := [3]int{7000, 7001, 7002}
	counter := &sync.Map{}
	ctx0, cancel0 := context.WithCancel(context.Background())
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	go func(ctx context.Context, port int) {
		startServer(ctx, port, counter)
	}(ctx0, ports[0])
	go func(ctx context.Context, port int) {
		startServer(ctx, port, counter)
	}(ctx1, ports[1])
	go func(ctx context.Context, port int) {
		startServer(ctx, port, counter)
	}(ctx2, ports[2])

	time.Sleep(1 * time.Second)

	b := NewRoundRobinBalancer()

	b.OnLeave(func(label string) {
		log.Println("client leave:", label)
	})
	defer b.Close()

	for i := 0; i < len(ports); i++ {
		client, err := rsocket.Connect().
			Transport(rsocket.TCPClient().SetHostAndPort("127.0.0.1", ports[i]).Build()).
			Start(context.Background())
		assert.NoError(t, err)
		_ = b.PutLabel(fmt.Sprintf("test-client-%d", ports[i]), client)
	}

	req := payload.NewString("foo", "bar")

	const n = 3
	wg := sync.WaitGroup{}
	wg.Add(n * len(ports))
	for i := 0; i < n*len(ports); i++ {
		c, ok := b.Next(context.Background())
		assert.True(t, ok, "get next client failed")
		c.RequestResponse(req).
			DoFinally(func(s rx.SignalType) {
				wg.Done()
			}).
			DoOnError(func(e error) {
				assert.Fail(t, "should never run here")
			}).
			SubscribeOn(scheduler.Parallel()).
			Subscribe(context.Background())
	}
	wg.Wait()

	counter.Range(func(key, value interface{}) bool {
		v := int(value.(*atomic.Int32).Load())
		assert.Equal(t, n, v)
		return true
	})

	ac0, ok := counter.Load(ports[0])
	assert.True(t, ok)

	amount0 := ac0.(*atomic.Int32).Load()

	// shutdown server 1
	cancel0()
	time.Sleep(100 * time.Millisecond)

	// then send a request
	c, ok := b.Next(context.Background())
	assert.True(t, ok, "get next client failed")
	_, err := c.RequestResponse(req).Block(context.Background())
	assert.NoError(t, err)

	var total int
	counter.Range(func(key, value interface{}) bool {
		total += int(value.(*atomic.Int32).Load())
		return true
	})
	assert.Equal(t, n*len(ports)+1, total)
	assert.Equal(t, int32(0), ac0.(*atomic.Int32).Load()-amount0)

	c, ok = b.Next(context.Background())
	assert.True(t, ok, "get next client failed")
	_, err = c.RequestResponse(req).Block(context.Background())
	assert.NoError(t, err)
	total++

	// shutdown server 2
	cancel1()
	time.Sleep(100 * time.Millisecond)

	const extra = 10

	for i := 0; i < extra; i++ {
		c, ok = b.Next(context.Background())
		assert.True(t, ok, "get next client failed")
		_, err = c.RequestResponse(req).Block(context.Background())
		assert.NoError(t, err)
	}
	total += 10

	requestedActual := 0
	counter.Range(func(key, value interface{}) bool {
		requestedActual += int(value.(*atomic.Int32).Load())
		return true
	})
	assert.Equal(t, total, requestedActual)

	cancel2()
	time.Sleep(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, ok = b.Next(ctx)
	assert.False(t, ok)
}
