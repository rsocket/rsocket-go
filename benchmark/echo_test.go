package benchmark

import (
	"bytes"
	"context"
	"fmt"
	"github.com/rsocket/rsocket-go"
	"github.com/stretchr/testify/assert"
	"log"
	_ "net/http/pprof"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var (
	host = "127.0.0.1"
	port = 8001
)

func TestEchoServe(t *testing.T) {
	if err := runEchoServer("127.0.0.1", port); err != nil {
		panic(err)
	}
}

func TestClient_Benchmark(t *testing.T) {
	cli, err := rsocket.Connect().
		SetupPayload(rsocket.NewPayloadString("你好", "世界")).
		Acceptor(func(socket rsocket.RSocket) rsocket.RSocket {
			return rsocket.NewAbstractSocket(
				rsocket.RequestResponse(func(payload rsocket.Payload) rsocket.Mono {
					log.Println("rcv reqresp from server:", payload)
					if bytes.Equal(payload.Data(), []byte("ping")) {
						return rsocket.JustMono(rsocket.NewPayloadString("pong", "from client"))
					}
					return rsocket.JustMono(payload)
				}),
			)
		}).
		Transport(fmt.Sprintf("%s:%d", host, port)).
		Start()
	if err != nil {
		t.Error(err)
	}
	defer func() {
		_ = cli.Close()
	}()

	begin := time.Now()
	totals := 500000
	wg := &sync.WaitGroup{}
	wg.Add(totals)
	data := []byte(strings.Repeat("A", 4096))

	var totalsRcv uint32
	for i := 0; i < totals; i++ {
		//log.Println("request:", i)
		// send 4k data
		md := []byte(fmt.Sprintf("benchmark_%d", i))
		cli.RequestResponse(rsocket.NewPayload(data, md)).
			DoFinally(func(ctx context.Context) {
				wg.Done()
			}).
			SubscribeOn(rsocket.ElasticScheduler()).
			Subscribe(context.Background(), func(ctx context.Context, item rsocket.Payload) {
				atomic.AddUint32(&totalsRcv, 1)
			})
	}
	wg.Wait()

	cost := time.Now().Sub(begin).Seconds()
	log.Println("--------------------------")
	log.Printf("QPS: %.2f\n", float64(totals)/cost)
	log.Println("--------------------------")
	assert.Equal(t, totals, int(totalsRcv))
}
