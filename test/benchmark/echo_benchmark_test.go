package benchmark

import (
	"bytes"
	"context"
	"fmt"
	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/stretchr/testify/assert"
	"log"
	_ "net/http/pprof"
	"strings"
	"sync"
	"testing"
	"time"
)

func doOnce(host string, port int, totals int) {
	wg := &sync.WaitGroup{}
	wg.Add(totals)
	data := []byte(strings.Repeat("A", 4096))
	md := []byte("benchmark_test")
	ctx := context.Background()
	log.Printf("CONN: %s:%d\n", host, port)
	clients := make([]rsocket.ClientSocket, totals)
	now := time.Now()
	for i := 0; i < totals; i++ {
		clients[i] = createClient(host, port)
	}
	log.Println("SETUP:", time.Now().Sub(now))
	now = time.Now()
	for _, client := range clients {
		client.RequestResponse(payload.New(data, md)).
			DoFinally(func(ctx context.Context, sig rx.SignalType) {
				wg.Done()
			}).
			SubscribeOn(rx.ElasticScheduler()).
			Subscribe(ctx)
	}
	wg.Wait()
	cost := time.Now().Sub(now)

	log.Println("TOTALS:", totals)
	log.Println("COST:", cost)
	log.Printf("QPS: %.2f\n", float64(totals)/cost.Seconds())
	time.Sleep(10 * time.Hour)
	for _, client := range clients {
		_ = client.Close()
	}
}

func TestClients_RequestResponse(t *testing.T) {
	log.Println("---------------")
	doOnce("127.0.0.1", 7878, 10000)
	//log.Println("---------------")
	//doOnce(host, 8000)
	//log.Println("---------------")
	//doOnce(host, 8001)
	//log.Println("---------------")
	//doOnce(host, 8000)
	//log.Println("---------------")
	//doOnce(host, 8001)
	//log.Println("---------------")
	//doOnce(host, 8000)
	//log.Println("---------------")
	//doOnce(host, 8001)
	//log.Println("---------------")
	//doOnce(host, 8000)
}

func TestClient_RequestResponse(t *testing.T) {
	client := createClient("127.0.0.1", 7878)
	defer func() {
		_ = client.Close()
	}()
	wg := &sync.WaitGroup{}
	n := 500000
	wg.Add(n)
	data := []byte(strings.Repeat("A", 4096))

	now := time.Now()
	ctx := context.Background()
	for i := 0; i < n; i++ {
		md := []byte(fmt.Sprintf("benchmark_test_%d", i))
		client.RequestResponse(payload.NewPooled(data, md)).
			SubscribeOn(rx.ElasticScheduler()).
			DoOnSuccess(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
				assert.Equal(t, data, elem.Data(), "data doesn't match")
				metadata, _ := elem.Metadata()
				assert.Equal(t, md, metadata, "metadata doesn't match")
				wg.Done()
			}).
			Subscribe(ctx)
	}
	wg.Wait()
	cost := time.Now().Sub(now)
	log.Println(n, "COST:", cost)
	log.Println(n, "QPS:", float64(n)/cost.Seconds())
}

func createClient(host string, port int) rsocket.ClientSocket {
	client, err := rsocket.Connect().
		SetupPayload(payload.NewString("你好", "世界")).
		Acceptor(func(socket rsocket.RSocket) rsocket.RSocket {
			return rsocket.NewAbstractSocket(
				rsocket.RequestResponse(func(p payload.Payload) rx.Mono {
					log.Println("rcv reqresp from server:", p)
					if bytes.Equal(p.Data(), []byte("ping")) {
						return rx.JustMono(payload.NewString("pong", "from client"))
					}
					return rx.JustMono(p)
				}),
			)
		}).
		Transport(fmt.Sprintf("%s:%d", host, port)).
		Start()
	if err != nil {
		panic(err)
	}
	return client
}
