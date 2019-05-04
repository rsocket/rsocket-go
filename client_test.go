package rsocket_test

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	. "github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/internal/logger"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	. "github.com/stretchr/testify/require"
)

var setupPayload payload.Payload
var connstr = "tcp://127.0.0.1:17878"

func init() {
	logger.SetLevel(logger.LevelDebug)
	setupPayload = payload.NewString("hello", "world")
}

func TestBlank(t *testing.T) {
	done := make(chan struct{})
	ctx := context.Background()
	cli, err := Connect().
		//Resume().
		OnClose(func() {
			close(done)
		}).
		SetupPayload(setupPayload).
		Transport(connstr).
		Start(ctx)
	NoError(t, err, "cannot create client")
	defer cli.Close()
	<-done
}

func TestNormal(t *testing.T) {
	done := make(chan struct{})
	cli, err := Connect().
		OnClose(func() {
			log.Println("closed 1")
		}).
		OnClose(func() {
			log.Println("closed 2")
		}).
		SetupPayload(setupPayload).
		Transport(connstr).
		Start(context.Background())
	NoError(t, err, "create client failed")
	defer func() {
		_ = cli.Close()
	}()
	do(t, cli)
	<-done
}

func TestResume(t *testing.T) {
	done := make(chan struct{})
	cli, err := Connect().
		SetupPayload(setupPayload).
		Resume(ResumeToken(func() []byte {
			return []byte{}
		})).
		OnClose(func() {
			close(done)
			log.Println("close done")
		}).
		Transport(connstr).
		Start(context.Background())
	NoError(t, err, "create client failed")
	defer func() {
		_ = cli.Close()
	}()

	tk := time.NewTicker(500 * time.Millisecond)

	var i int32

L:
	for {
		select {
		case <-done:
			break L
		case <-tk.C:
			cli.RequestResponse(payload.NewString(fmt.Sprintf("foo_%04d", i), "bar")).
				DoOnSuccess(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
					log.Println(">>>>>>>", elem)
				}).
				Subscribe(context.Background())
		}
		i++
	}
}

func do(t *testing.T, cli Client) {
	cli.RequestResponse(payload.NewString("foo", "bar")).
		DoOnSuccess(func(ctx context.Context, s rx.Subscription, elem payload.Payload) {
			log.Println("got:", elem)
		}).
		Subscribe(context.Background())
}
