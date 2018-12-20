package rsocket

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"log"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestClient_Benchmark(t *testing.T) {
	host := "127.0.0.1"
	port := 8000

	server := runEchoServer(host, port)

	go server.Serve()

	ctx, cancel := context.WithCancel(context.Background())

	defer func() {
		cancel()
		_ = server.Close()
	}()

	cli, err := NewClient(
		WithTCPTransport(host, port),
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
	totals := 10
	wg := &sync.WaitGroup{}
	wg.Add(totals)
	for i := 0; i < totals; i++ {
		send := CreatePayloadString(strings.Repeat("A", 128), fmt.Sprintf("benchmark_%d", i))
		err := cli.RequestResponse(send, func(res Payload, err error) {
			wg.Done()
			assert.Equal(t, res.Data(), send.Data(), "data doesn't match")
			assert.Equal(t, res.Metadata(), send.Metadata(), "metadata doesn't match")
		})
		assert.NoError(t, err)
	}
	wg.Wait()
}

func runEchoServer(host string, port int) *Server {
	server, err := NewServer(
		WithTCPServerTransport(fmt.Sprintf("%s:%d", host, port)),
		WithAcceptor(func(setup SetupPayload, rs *RSocket) (err error) {
			rs.HandleRequestResponse(func(req Payload) (res Payload, err error) {
				// just echo
				return req, nil
			})
			return nil
		}),
	)
	if err != nil {
		panic(err)
	}
	return server
}
