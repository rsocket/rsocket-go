package rsocket

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
	"time"
)

func newMockClient() ClientSocket {
	socket, err := Connect().
		SetupPayload(NewPayloadString("hello", "world")).
		MetadataMimeType("application/json").
		DataMimeType("application/json").
		KeepAlive(3*time.Second, 2*time.Second, 3).
		Acceptor(func(socket RSocket) RSocket {
			return NewAbstractSocket(
				RequestResponse(func(payload Payload) Mono {
					return JustMono(NewPayloadString("foo", "bar"))
				}),
			)
		}).
		Transport("127.0.0.1:8001").
		Start()
	if err != nil {
		panic(err)
	}
	return socket
}

func TestClient_RequestResponse(t *testing.T) {
	socket := newMockClient()
	defer func() {
		_ = socket.Close()
	}()
	socket.RequestResponse(NewPayloadString("hello", "world")).
		DoOnError(func(ctx context.Context, err error) {
			log.Println(err)
		}).
		DoOnCancel(func(ctx context.Context) {
			log.Println("oops...it's canceled")
		}).
		Subscribe(context.Background(), func(ctx context.Context, item Payload) {
			log.Println("rcv:", item)
			assert.Equal(t, "hello", string(item.Data()))
			assert.Equal(t, "world", string(item.Metadata()))
		})
	time.Sleep(3*time.Second)
}

func TestClient_Mock(t *testing.T) {
	socket := newMockClient()
	defer func() {
		_ = socket.Close()
	}()
	socket.RequestResponse(NewPayloadString("see", "you")).
		DoFinally(func(ctx context.Context) {
			log.Println("final reqresp")
		}).
		DoOnError(func(ctx context.Context, err error) {
			log.Println("reqresp error:", err)
		}).
		Subscribe(context.Background(), func(ctx context.Context, item Payload) {
			log.Println("WAHAHA:", item)
		})
	socket.RequestStream(NewPayloadString("aaa", "bbb")).
		DoFinally(func(ctx context.Context) {
			log.Println("finish")
		}).
		Subscribe(context.Background(), func(ctx context.Context, item Payload) {
			log.Println("stream:", item)
		})

	socket.
		RequestChannel(NewFlux(func(ctx context.Context, emitter Emitter) {
			for i := 0; i < 5; i++ {
				emitter.Next(NewPayloadString(fmt.Sprintf("hello_%d", i), "from golang"))
			}
			emitter.Complete()
		})).
		Subscribe(context.Background(), func(ctx context.Context, item Payload) {
			log.Println("channel:", item)
		})
}
