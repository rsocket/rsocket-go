package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"flag"
	"log"
	"sync"
	"time"

	"github.com/pkg/profile"
	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/mono"
	"go.uber.org/atomic"
)

var tp transport.ClientTransporter

var (
	n           int
	payloadSize int
	mtu         int
)

func init() {
	flag.IntVar(&n, "n", 100*10000, "request amount.")
	flag.IntVar(&payloadSize, "size", 1024, "payload data size.")
	flag.IntVar(&mtu, "mtu", 0, "mut size, zero means disabled.")
	flag.Parse()
	tp = rsocket.TCPClient().SetHostAndPort("127.0.0.1", 7878).Build()
}

func main() {
	defer profile.Start(profile.MemProfile, profile.ProfilePath(".")).Stop()

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
	log.Printf("total:\t\t%d\n", n)
	log.Printf("cost:\t\t%s\n", cost)
	log.Printf("qps:\t\t%.02f\n", float64(n)/cost.Seconds())
	if errCount.Load() > 0 {
		log.Printf("error:\t\t%d\n", errCount.Load())
	}
	throughputInMB := float64(n*(1024+9)) / float64(1024*1024) / cost.Seconds()
	log.Printf("throughput(r/w):\t%.02fMB/s, %.02fMB/s\n", throughputInMB, throughputInMB)
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
