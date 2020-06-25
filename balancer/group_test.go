package balancer_test

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"log"
	"testing"
	"time"

	. "github.com/rsocket/rsocket-go"
	. "github.com/rsocket/rsocket-go/balancer"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx/mono"
	"github.com/stretchr/testify/require"
)

var tp Transporter

func init() {
	tp = Tcp().HostAndPort("127.0.0.1", 7878).Build()
}

func ExampleNewGroup() {
	group := NewGroup(func() Balancer {
		return NewRoundRobinBalancer()
	})
	defer func() {
		_ = group.Close()
	}()
	// Create a broker with resume.
	err := Receive().
		Resume(WithServerResumeSessionDuration(10 * time.Second)).
		Acceptor(func(setup payload.SetupPayload, sendingSocket CloseableRSocket) (RSocket, error) {
			// Register service using Setup Metadata as service ID.
			if serviceID, ok := setup.MetadataUTF8(); ok {
				group.Get(serviceID).Put(sendingSocket)
			}
			// Proxy requests by group.
			return NewAbstractSocket(RequestResponse(func(msg payload.Payload) mono.Mono {
				requestServiceID, ok := msg.MetadataUTF8()
				if !ok {
					panic(errors.New("missing service ID in metadata"))
				}
				log.Println("[broker] redirect request to service", requestServiceID)
				return group.Get(requestServiceID).MustNext(context.Background()).RequestResponse(msg)
			})), nil
		}).
		Transport(tp).
		Serve(context.Background())
	if err != nil {
		panic(err)
	}
}

func TestServiceSubscribe(t *testing.T) {
	// Init broker and service.
	go ExampleNewGroup()

	// Waiting broker up by sleeping 200 ms.
	time.Sleep(200 * time.Millisecond)

	// Deploy MD5 service.
	go func() {
		done := make(chan struct{})
		cli, err := Connect().
			OnClose(func(err error) {
				close(done)
			}).
			SetupPayload(payload.NewString("This is a Service Publisher!", "md5")).
			Acceptor(func(socket RSocket) RSocket {
				return NewAbstractSocket(RequestResponse(func(msg payload.Payload) mono.Mono {
					result := payload.NewString(fmt.Sprintf("%02x", md5.Sum(msg.Data())), "MD5 RESULT")
					log.Println("[publisher] accept MD5 request:", msg.DataUTF8())
					return mono.Just(result)
				}))
			}).
			Transport(tp).
			Start(context.Background())
		if err != nil {
			panic(err)
		}
		defer func() {
			_ = cli.Close()
		}()
		<-done
	}()

	// Create a client and request md5 service.
	cli, err := Connect().
		SetupPayload(payload.NewString("This is a Subscriber", "")).
		Transport(tp).
		Start(context.Background())
	require.NoError(t, err, "create client failed")
	defer func() {
		_ = cli.Close()
		time.Sleep(200 * time.Millisecond)
	}()
	_, err = cli.RequestResponse(payload.NewString("Hello World!", "md5")).
		DoOnSuccess(func(elem payload.Payload) {
			log.Println("[subscriber] receive MD5 response:", elem.DataUTF8())
			require.Equal(t, "ed076287532e86365e841e92bfc50d8c", elem.DataUTF8(), "bad md5")
		}).
		Block(context.Background())
	require.NoError(t, err, "request failed")
}
