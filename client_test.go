package rsocket

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"
)

func TestClient_Mock(t *testing.T) {
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
		Transport("127.0.0.1:8000").
		Start()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = socket.Close()
	}()
	socket.RequestResponse(NewPayloadString("see", "you")).
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
