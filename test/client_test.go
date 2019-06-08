package test

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/logger"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/stretchr/testify/assert"
)

var client rsocket.Client

func init() {
	logger.SetLevel(logger.LevelDebug)
	const connStr = "tcp://127.0.0.1:7878"
	acceptor := rsocket.NewAbstractSocket(
		rsocket.MetadataPush(func(payload payload.Payload) {
			log.Println("rcv MetadataPush:", payload)
		}),
		rsocket.FireAndForget(func(msg payload.Payload) {
			log.Println("rcv FNF:", msg)
		}),
		rsocket.RequestResponse(func(msg payload.Payload) rx.Mono {
			return rx.JustMono(msg)
		}),
		rsocket.RequestStream(func(msg payload.Payload) rx.Flux {
			d := msg.DataUTF8()
			m, _ := msg.MetadataUTF8()
			totals, _ := strconv.Atoi(m)
			return rx.Range(0, totals).
				Map(func(n int) payload.Payload {
					return payload.NewString(fmt.Sprintf("%s_%d", d, n), m)
				})
		}),
		rsocket.RequestChannel(func(msgs rx.Publisher) rx.Flux {
			return rx.ToFlux(msgs)
		}))

	go func() {
		err := rsocket.Receive().
			Acceptor(func(setup payload.SetupPayload, sendingSocket rsocket.CloseableRSocket) rsocket.RSocket {
				return acceptor
			}).
			Transport(connStr).
			Serve()
		panic(err)
	}()

	// TODO: ugly code for waiting for server serve.
	time.Sleep(1 * time.Second)

	socket, err := rsocket.Connect().
		SetupPayload(payload.NewString("hello", "world")).
		MetadataMimeType("application/json").
		DataMimeType("application/json").
		KeepAlive(3*time.Second, 2*time.Second, 3).
		Acceptor(func(socket rsocket.RSocket) rsocket.RSocket {
			return acceptor
		}).
		Transport(connStr).
		Start(context.Background())
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
			assert.Equal(t, "hello", elem.DataUTF8())
			metadata, _ := elem.MetadataUTF8()
			assert.Equal(t, "world", metadata)
		}).
		Subscribe(context.Background())
}

func TestClient_RequestStream(t *testing.T) {
	done := make(chan struct{})

	var totals int

	c := common.RandIntn(10) + 1

	client.RequestStream(payload.NewString("hello", fmt.Sprintf("%d", c))).
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
			assert.Equal(t, fmt.Sprintf("hello_%d", totals), elem.DataUTF8(), "bad data")
			metadata, _ := elem.MetadataUTF8()
			assert.Equal(t, fmt.Sprintf("%d", c), metadata, "bad metadata")
			totals++
			s.Request(1)
		}).
		DoOnSubscribe(func(ctx context.Context, s rx.Subscription) {
			s.Request(1)
		}).
		Subscribe(context.Background())
	<-done
}

func TestClient_RequestChannel(t *testing.T) {
	done := make(chan struct{})
	sending := rx.Range(0, 3).Map(func(n int) payload.Payload {
		return payload.NewString(fmt.Sprintf("D_%d", n), "C")
	})
	client.
		RequestChannel(sending).
		DoFinally(func(ctx context.Context, sig rx.SignalType) {
			close(done)
		}).
		DoOnNext(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
			//log.Println("next:", elem)
		}).
		Subscribe(context.Background())
	<-done
}
