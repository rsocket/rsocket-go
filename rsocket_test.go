package rsocket

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"testing"
	"time"
)

func runEchoServer(ctx context.Context) {
	server, err := NewServer(
		WithTransportTCP("127.0.0.1:8000"),
		WithAcceptor(func(setup SetupPayload, rs *RSocket) (err error) {
			log.Printf("SETUP: version=%s, data=%s, metadata=%s\n", setup.Version(), string(setup.Data()), string(setup.Metadata()))
			return nil
		}),
		WithFireAndForget(func(req Payload) error {
			log.Println("GOT FNF:", req)
			return nil
		}),
		WithRequestResponseHandler(func(req Payload) (res Payload, err error) {
			// just echo
			return req, nil
		}),
		WithRequestStreamHandler(func(req Payload, emitter Emitter) {
			totals := 1000
			for i := 0; i < totals; i++ {
				payload := CreatePayloadString(fmt.Sprintf("%d", i), "")
				if err := emitter.Next(payload); err != nil {
					log.Println("process stream failed:", err)
				}
			}
			payload := CreatePayloadString(fmt.Sprintf("%d", totals), "")
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

	cli, err := NewClient(
		WithTCPTransport("127.0.0.1", 8000),
		WithSetupPayload([]byte("你好"), []byte("世界")),
		WithKeepalive(2*time.Second, 3*time.Second, 3),
		WithMetadataMimeType("application/binary"),
		WithDataMimeType("application/binary"),
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
	totals := 100000
	wg := &sync.WaitGroup{}
	wg.Add(totals)
	for i := 0; i < totals; i++ {
		// send 4k data
		send := CreatePayloadString(strings.Repeat("A", 4096), fmt.Sprintf("benchmark_%d", i))
		if err := cli.RequestResponse(send, func(res Payload, err error) {
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
