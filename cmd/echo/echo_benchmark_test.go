package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	_ "net/http/pprof"
	"sync"
	"testing"
	"time"

	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/mono"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_RequestResponse(t *testing.T) {
	client, err := createClient(ListenAt)
	require.NoError(t, err, "bad client")
	defer func() {
		_ = client.Close()
	}()
	wg := &sync.WaitGroup{}
	n := 50 * 10000
	wg.Add(n)
	data := []byte(common.RandAlphanumeric(1024))

	now := time.Now()
	ctx := context.Background()

	sub := rx.NewSubscriber(
		rx.OnNext(func(input payload.Payload) {
			assert.Equal(t, data, input.Data(), "data doesn't match")
			//m2, _ := elem.MetadataUTF8()
			//assert.Equal(t, m1, m2, "metadata doesn't match")
			wg.Done()
		}),
	)

	for i := 0; i < n; i++ {
		m1 := []byte(fmt.Sprintf("benchmark_test_%d", i))
		client.RequestResponse(payload.New(data, m1)).SubscribeOn(scheduler.Elastic()).SubscribeWith(ctx, sub)
	}
	wg.Wait()
	cost := time.Since(now)
	log.Println(n, "COST:", cost)
	log.Println(n, "QPS:", float64(n)/cost.Seconds())

}

func createClient(uri string) (rsocket.Client, error) {
	return rsocket.Connect().
		//Fragment(1024).
		//Resume().
		SetupPayload(payload.NewString("你好", "世界")).
		Acceptor(func(socket rsocket.RSocket) rsocket.RSocket {
			return rsocket.NewAbstractSocket(
				rsocket.RequestResponse(func(p payload.Payload) mono.Mono {
					log.Println("rcv reqresp from server:", p)
					if bytes.Equal(p.Data(), []byte("ping")) {
						return mono.Just(payload.NewString("pong", "from client"))
					}
					return mono.Just(p)
				}),
			)
		}).
		Transport(uri).
		Start(context.Background())
}
