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
		Transport("127.0.0.1:8000").
		Start()
	if err != nil {
		log.Fatal(err)
	}
	return socket
}

func TestClient_KeepAlive(t *testing.T) {
	cli := newMockClient()

	defer cli.Close()

	time.Sleep(1 * time.Hour)

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
		Subscribe(context.Background(), OnNext(func(ctx context.Context, sub Subscription, item Payload) {
			log.Println("rcv:", item)
			assert.Equal(t, "hello", string(item.Data()))
			assert.Equal(t, "world", string(item.Metadata()))
		}))
}

func TestClient_RequestStream(t *testing.T) {
	socket := newMockClient()
	defer func() {
		_ = socket.Close()
	}()
	done := make(chan struct{})
	socket.RequestStream(NewPayloadString("hello", "50")).
		DoFinally(func(ctx context.Context, sig SignalType) {
			close(done)
		}).
		DoOnError(func(ctx context.Context, err error) {
			log.Println(err)
			close(done)
		}).
		DoOnCancel(func(ctx context.Context) {
			log.Println("oops...it's canceled")
		}).
		LimitRate(3).
		Subscribe(context.Background(), OnNext(func(ctx context.Context, sub Subscription, payload Payload) {
			log.Println("rcv:", payload)
		}))
	<-done
}

func TestClient_MetadataPush(t *testing.T) {
	socket := newMockClient()
	defer func() {
		_ = socket.Close()
	}()
	socket.MetadataPush(NewPayloadString("hello", "world"))
	time.Sleep(10 * time.Second)
}

func TestClient_FireAndForget(t *testing.T) {
	socket := newMockClient()
	defer func() {
		_ = socket.Close()
	}()
	socket.FireAndForget(NewPayloadString("hello", "world"))
}

func TestClient_RequestChannel(t *testing.T) {
	socket := newMockClient()
	defer func() {
		_ = socket.Close()
	}()

	done := make(chan struct{})
	socket.
		RequestChannel(NewFlux(func(ctx context.Context, emitter Producer) {
			for i := 0; i < 10; i++ {
				emitter.Next(NewPayloadString("h", "b"))
			}
			emitter.Complete()
		})).
		DoFinally(func(ctx context.Context, sig SignalType) {
			close(done)
		}).
		Subscribe(context.Background(), OnNext(func(ctx context.Context, sub Subscription, payload Payload) {
			log.Println("next:", payload)
		}))
	<-done
}

func TestClient_Mock(t *testing.T) {
	socket := newMockClient()
	defer func() {
		_ = socket.Close()
	}()
	socket.RequestResponse(NewPayloadString("see", "you")).
		DoFinally(func(ctx context.Context, sig SignalType) {
			log.Println("final reqresp")
		}).
		DoOnError(func(ctx context.Context, err error) {
			log.Println("reqresp error:", err)
		}).
		Subscribe(context.Background(), OnNext(func(ctx context.Context, sub Subscription, payload Payload) {
			log.Println("WAHAHA:", payload)
		}))
	socket.RequestStream(NewPayloadString("aaa", "bbb")).
		DoFinally(func(ctx context.Context, sig SignalType) {
			log.Println("finish")
		}).
		Subscribe(context.Background(), OnNext(func(ctx context.Context, sub Subscription, payload Payload) {
			log.Println("stream:", payload)
		}))

	socket.
		RequestChannel(NewFlux(func(ctx context.Context, emitter Producer) {
			for i := 0; i < 5; i++ {
				emitter.Next(NewPayloadString(fmt.Sprintf("hello_%d", i), "from golang"))
			}
			emitter.Complete()
		})).
		Subscribe(context.Background(), OnNext(func(ctx context.Context, sub Subscription, payload Payload) {
			log.Println("channel:", payload)
		}))
}
