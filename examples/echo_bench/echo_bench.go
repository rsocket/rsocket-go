package main

import (
	"bytes"
	"context"
	"flag"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/mono"
)

var tp rsocket.Transporter

func init() {
	flag.Parse()
	tp = rsocket.Tcp().HostAndPort("127.0.0.1", 7878).Build()
	rand.Seed(time.Now().UnixNano())
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
	ctx := context.Background()

	sub := rx.NewSubscriber(
		rx.OnNext(func(input payload.Payload) {
			//m2, _ := elem.MetadataUTF8()
			//assert.Equal(t, m1, m2, "metadata doesn't match")
			wg.Done()
		}),
	)

	for i := 0; i < n; i++ {
		client.RequestResponse(payload.New(data, nil)).SubscribeOn(scheduler.Elastic()).SubscribeWith(ctx, sub)
	}
	wg.Wait()
	cost := time.Since(now)
	log.Println(n, "COST:", cost)
	log.Println(n, "QPS:", float64(n)/cost.Seconds())
}

func createClient(mtu int) (rsocket.Client, error) {
	return rsocket.Connect().
		Fragment(mtu).
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
		Transport(tp).
		Start(context.Background())
}
