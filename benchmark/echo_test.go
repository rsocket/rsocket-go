package benchmark

import (
	"context"
	"fmt"
	"github.com/rsocket/rsocket-go"
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

func TestEchoServe(t *testing.T) {
	server := runEchoServer("127.0.0.1", port)
	defer func() {
		_ = server.Close()
	}()
	_ = server.Serve()
}

func TestClient_Benchmark(t *testing.T) {
	//server := runEchoServer(host, port)
	//go server.Serve()
	//time.Sleep(300 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())

	defer func() {
		cancel()
		//_ = server.Close()
	}()

	cli, err := rsocket.NewClient(
		rsocket.WithTCPTransport(host, port),
		rsocket.WithSetupPayload([]byte("你好"), []byte("世界")),
		rsocket.WithKeepalive(2*time.Second, 3*time.Second, 3),
		rsocket.WithMetadataMimeType("application/binary"),
		rsocket.WithDataMimeType("application/binary"),
	)
	if err != nil {
		t.Error(err)
	}
	defer func() {
		if err := cli.Close(); err != nil {
			log.Println("close client failed:", err)
		}
	}()
	if err := cli.Start(ctx); err != nil {
		t.Error(err)
	}
	begin := time.Now()
	totals := 200000
	wg := &sync.WaitGroup{}
	wg.Add(totals)
	data := []byte(strings.Repeat("A", 4096))
	for i := 0; i < totals; i++ {
		//log.Println("request:", i)
		// send 4k data
		md := []byte(fmt.Sprintf("benchmark_%d", i))
		err := cli.RequestResponse(data, md, func(res rsocket.Payload, err error) {
			//if !bytes.Equal(res.Data(), data) {
			//	t.Error("data doesn't match:", string(res.Data()), string(data))
			//}
			//if !bytes.Equal(res.Metadata(), md) {
			//	t.Error("metadata doesn't match:", string(res.Metadata()), string(md))
			//}
			wg.Done()
		})
		if err != nil {
			t.Error(err)
		}
	}
	wg.Wait()

	cost := time.Now().Sub(begin).Seconds()
	log.Println("--------------------------")
	log.Printf("QPS: %.2f\n", float64(totals)/cost)
	log.Println("--------------------------")
}
