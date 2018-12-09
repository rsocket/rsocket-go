package benchmark

import (
	"bytes"
	"context"
	"fmt"
	"github.com/jjeffcaii/go-rsocket"
	"log"
	"strings"
	"sync"
	"testing"
	"time"
)

func runEchoServer(ctx context.Context) {
	server, err := rsocket.NewServer(
		rsocket.WithTransportTCP("127.0.0.1:8000"),
		rsocket.WithAcceptor(func(setup rsocket.SetupPayload, rs *rsocket.RSocket) (err error) {
			log.Printf("SETUP: version=%s, data=%s, metadata=%s\n", setup.Version(), string(setup.Data()), string(setup.Metadata()))
			return nil
		}),
		rsocket.WithFireAndForget(func(req rsocket.Payload) error {
			log.Println("GOT FNF:", req)
			return nil
		}),
		rsocket.WithRequestResponseHandler(func(req rsocket.Payload) (res rsocket.Payload, err error) {
			// just echo
			return req, nil
		}),
		rsocket.WithRequestStreamHandler(func(req rsocket.Payload, emitter rsocket.Emitter) {
			totals := 1000
			for i := 0; i < totals; i++ {
				payload := rsocket.CreatePayloadString(fmt.Sprintf("%d", i), "")
				if err := emitter.Next(payload); err != nil {
					log.Println("process stream failed:", err)
				}
			}
			payload := rsocket.CreatePayloadString(fmt.Sprintf("%d", totals), "")
			if err := emitter.Complete(payload); err != nil {
				log.Println("process stream failed:", err)
			}
		}),
	)
	if err != nil {
		panic(err)
	}
	go func(ctx context.Context) {
		if err := server.Start(ctx); err != nil {
			log.Println("server stopped:", err)
		}
	}(ctx)
}

func TestClient_Benchmark(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	runEchoServer(ctx)

	defer cancel()

	time.Sleep(1 * time.Second)

	cli, err := rsocket.NewClient(
		rsocket.WithTCPTransport("127.0.0.1", 8000),
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
	totals := 10000
	wg := &sync.WaitGroup{}
	wg.Add(totals)
	for i := 0; i < totals; i++ {
		// send 4k data
		send := rsocket.CreatePayloadString(strings.Repeat("A", 4096), fmt.Sprintf("benchmark_%d", i))
		if err := cli.RequestResponse(send, func(res rsocket.Payload, err error) {
			if !bytes.Equal(res.Data(), send.Data()) {
				t.Error("data doesn't match")
			}
			if !bytes.Equal(res.Metadata(), send.Metadata()) {
				t.Error("metadata doesn't match")
			}
			wg.Done()
		}); err != nil {
			t.Error(err)
		}
	}
	wg.Wait()
	cost := (time.Now().UnixNano() - begin.UnixNano()) / 1e6
	log.Println("--------------------------")
	log.Println("QPS:", 1000*totals/int(cost))
	log.Println("--------------------------")
}
