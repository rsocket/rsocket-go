package test

import (
	"context"
	"fmt"
	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/common/logger"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

var client rsocket.ClientSocket

func init() {
	socket, err := rsocket.Connect().
		SetupPayload(payload.NewString("hello", "world")).
		MetadataMimeType("application/json").
		DataMimeType("application/json").
		KeepAlive(3*time.Second, 2*time.Second, 3).
		Acceptor(func(socket rsocket.RSocket) rsocket.RSocket {
			return rsocket.NewAbstractSocket(
				rsocket.RequestResponse(func(msg payload.Payload) rx.Mono {
					return rx.JustMono(payload.NewString("foo", "bar"))
				}),
			)
		}).
		Transport("127.0.0.1:7878").
		Start()
	if err != nil {
		log.Fatal(err)
	}
	client = socket
	logger.Infof("+++++ CONNECT SUCCESS +++++\n")

	done := make(chan bool, 1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println(sig)
		done <- true
	}()

	go func() {
		<-done
		_ = client.Close()
		log.Println("socket closed")
	}()
}

func TestClient_MetadataPush(t *testing.T) {
	client.MetadataPush(payload.NewString("hello", "world"))
}

func TestClient_FireAndForget(t *testing.T) {
	client.FireAndForget(payload.NewString("hello", "world"))
}

func TestClient_RequestResponse(t *testing.T) {
	client.RequestResponse(payload.NewString("hello", "world")).
		DoOnError(func(ctx context.Context, err error) {
			log.Println("oops...", err)
		}).
		DoOnCancel(func(ctx context.Context) {
			log.Println("oops...it's canceled")
		}).
		DoOnSuccess(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
			log.Println("rcv:", elem)
			assert.Equal(t, "hello", string(elem.Data()))
			assert.Equal(t, "world", string(elem.Metadata()))
		}).
		Subscribe(context.Background())
}

func TestClient_RequestStream(t *testing.T) {
	done := make(chan struct{})

	var totals int

	c := 7

	client.RequestStream(payload.NewString("hello", fmt.Sprintf("%d", c))).
		LimitRate(3).
		DoFinally(func(ctx context.Context, sig rx.SignalType) {
			close(done)
		}).
		DoOnError(func(ctx context.Context, err error) {
			log.Println("oops...", err)
		}).
		DoOnCancel(func(ctx context.Context) {
			log.Println("oops...it's canceled")
		}).
		DoOnNext(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
			time.Sleep(500 * time.Millisecond)
			log.Println("rcv:", elem)
			assert.Equal(t, fmt.Sprintf("hello_%d", totals), string(elem.Data()), "bad data")
			assert.Equal(t, fmt.Sprintf("%d", c), string(elem.Metadata()), "bad metadata")
			totals++
		}).
		Subscribe(context.Background())
	<-done
}

func TestClient_RequestChannel(t *testing.T) {
	done := make(chan struct{})
	client.
		RequestChannel(rx.NewFlux(func(ctx context.Context, emitter rx.Producer) {
			for i := 0; i < 10; i++ {
				_ = emitter.Next(payload.NewString("h", "b"))
			}
			emitter.Complete()
		})).
		DoFinally(func(ctx context.Context, sig rx.SignalType) {
			close(done)
		}).
		DoOnNext(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
			log.Println("next:", elem)
		}).
		Subscribe(context.Background())
	<-done
}
