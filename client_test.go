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

// Test client with Echo Server: (java tcp server as below)
//     RSocketFactory.receive()
//        .acceptor((setup, sendingSocket) -> Mono.just(new AbstractRSocket() {
//          @Override
//          public Mono<Payload> requestResponse(Payload payload) {
//            payload.release();
//            return Mono.just(DefaultPayload.create(payload));
//          }
//        }))
//        .transport(TcpServerTransport.create("127.0.0.1", 8000))
//        .start()
//        .block()
//        .onClose()
//        .block();
func TestNewClient(t *testing.T) {
	cli, err := NewClient(
		WithTCPTransport("127.0.0.1", 8000),
		WithSetupPayload([]byte("hello"), nil),
		WithKeepalive(2*time.Second, 3*time.Second, 3),
	)
	if err != nil {
		t.Error(err)
	}

	defer func() {
		if err := cli.Close(); err != nil {
			log.Println("close client failed:", err)
		}
	}()

	ctx := context.Background()
	if err := cli.Start(ctx); err != nil {
		t.Error(err)
	}
	begin := time.Now()
	totals := 100
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
	log.Println("QPS:", 1000*totals/int(cost))
}
