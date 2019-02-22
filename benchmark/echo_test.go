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
	"testing"
	"time"
)

var (
	host = "127.0.0.1"
	port = 8001
)

func createClient(host string, port int) rsocket.ClientSocket {
	client, err := rsocket.Connect().
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
		panic(err)
	}
	return client
}

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
		client.RequestResponse(rsocket.NewPayload(data, md)).
			DoFinally(func(ctx context.Context) {
				wg.Done()
			}).
			SubscribeOn(rsocket.ElasticScheduler()).
			Subscribe(ctx, func(ctx context.Context, item rsocket.Payload) {
			})
	}
	wg.Wait()
	cost := time.Now().Sub(now)

	log.Println("TOTALS:", totals)
	log.Println("COST:", cost)
	log.Printf("QPS: %.2f\n", float64(totals)/cost.Seconds())
	time.Sleep(10 * time.Second)
	//for _, client := range clients {
	//	_ = client.Close()
	//}
}

func TestClients_RequestResponse(t *testing.T) {
	log.Println("---------------")
	doOnce(host, 8001, 5000)
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
	client := createClient(host, port)
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
		client.RequestResponse(rsocket.NewPayload(data, md)).
			DoFinally(func(ctx context.Context) {
				wg.Done()
			}).
			SubscribeOn(rsocket.ElasticScheduler()).
			Subscribe(ctx, func(ctx context.Context, item rsocket.Payload) {
				assert.Equal(t, data, item.Data(), "data doesn't match")
				assert.Equal(t, md, item.Metadata(), "metadata doesn't match")
			})
	}
	wg.Wait()
	cost := time.Now().Sub(now)
	log.Println(n, "COST:", cost)
	log.Println(n, "QPS:", float64(n)/cost.Seconds())
}
