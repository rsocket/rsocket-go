package benchmark

import (
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
			DoFinally(func(ctx context.Context, sig rsocket.SignalType) {
				wg.Done()
			}).
			SubscribeOn(rsocket.ElasticScheduler()).
			Subscribe(ctx, rsocket.OnNext(func(ctx context.Context, sub rsocket.Subscription, payload rsocket.Payload) {
			}))
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
	doOnce("127.0.0.1", 8001, 10000)
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
	client := createClient("127.0.0.1", 8001)
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
			//DoFinally(func(ctx context.Context, sig rsocket.SignalType) {
			//
			//}).
			SubscribeOn(rsocket.ElasticScheduler()).
			Subscribe(ctx, rsocket.OnNext(func(ctx context.Context, sub rsocket.Subscription, payload rsocket.Payload) {
				assert.Equal(t, data, payload.Data(), "data doesn't match")
				assert.Equal(t, md, payload.Metadata(), "metadata doesn't match")
				wg.Done()
			}))
	}
	wg.Wait()
	cost := time.Now().Sub(now)
	log.Println(n, "COST:", cost)
	log.Println(n, "QPS:", float64(n)/cost.Seconds())
}
