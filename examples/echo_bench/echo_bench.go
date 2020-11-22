package main

import (
	"bytes"
	"context"
	"flag"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"time"

	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/mono"
	"go.uber.org/atomic"
)

var tp transport.ClientTransporter

func init() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	tp = rsocket.TCPClient().SetHostAndPort("127.0.0.1", 7878).Build()

	go func() {
		log.Println(http.ListenAndServe(":5555", nil))
	}()
}

func main() {
	var (
		n           int
		payloadSize int
		mtu         int
	)
	flag.IntVar(&n, "n", 100*10000, "request amount.")
	flag.IntVar(&payloadSize, "size", 1024, "payload data size.")
	flag.IntVar(&mtu, "mtu", 0, "mut size, zero means disabled.")

	client, err := createClient(mtu)
	if err != nil {
		panic(err)
	}
	defer client.Close()
	wg := &sync.WaitGroup{}

	wg.Add(n)
	data := make([]byte, payloadSize)
	rand.Read(data)

	now := time.Now()

	errCount := atomic.NewInt32(0)

	sub := rx.NewSubscriber(
		rx.OnNext(func(input payload.Payload) error {
			//m2, _ := elem.MetadataUTF8()
			//assert.Equal(t, m1, m2, "metadata doesn't match")
			return nil
		}),
		rx.OnError(func(e error) {
			errCount.Inc()
			wg.Done()
		}),
		rx.OnComplete(func() {
			wg.Done()
		}),
	)

	request := payload.New(data, nil)
	for i := 0; i < n; i++ {
		client.RequestResponse(request).SubscribeWith(context.Background(), sub)
	}
	wg.Wait()
	cost := time.Since(now)
	log.Println("TOTAL:", n)
	log.Println("COST:", cost)
	log.Printf("QPS: %.02f\n", float64(n)/cost.Seconds())
	log.Println("FAILED:", errCount.Load())
}

func createClient(mtu int) (rsocket.Client, error) {
	return rsocket.Connect().
		Fragment(mtu).
		OnClose(func(err error) {
			log.Println("*** disconnected ***", common.CountBorrowed())
		}).
		SetupPayload(payload.NewString("你好", "世界")).
		Acceptor(func(socket rsocket.RSocket) rsocket.RSocket {
			return rsocket.NewAbstractSocket(
				rsocket.RequestResponse(func(p payload.Payload) mono.Mono {
					log.Println("receive request from server:", p)
					if bytes.Equal(p.Data(), []byte("ping")) {
						return mono.Just(payload.NewString("pong", "from client"))
					}
					return mono.Just(p)
				}),
			)
		}).
		Transport(tp).
		Start(context.Background())
}
