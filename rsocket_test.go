package rsocket_test

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	. "github.com/rsocket/rsocket-go"
	. "github.com/rsocket/rsocket-go/payload"
	. "github.com/rsocket/rsocket-go/rx"
	"github.com/stretchr/testify/require"
)

func Example() {
	// Serve a server
	err := Receive().
		Acceptor(func(setup SetupPayload, sendingSocket CloseableRSocket) RSocket {
			return NewAbstractSocket(
				RequestResponse(func(msg Payload) Mono {
					log.Println("incoming request:", msg)
					return JustMono(NewString("Pong", time.Now().String()))
				}),
			)
		}).
		Transport("tcp://127.0.0.1:7878").
		Serve(context.Background())
	if err != nil {
		panic(err)
	}

	// Connect to a server.
	cli, err := Connect().
		SetupPayload(NewString("Hello World", "From Golang")).
		Transport("tcp://127.0.0.1:7878").
		Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = cli.Close()
	}()
	cli.RequestResponse(NewString("Ping", time.Now().String())).
		DoOnSuccess(func(ctx context.Context, s Subscription, elem Payload) {
			log.Println("incoming response:", elem)
		}).
		Subscribe(context.Background())
}

func ExampleReceive() {
	err := Receive().
		Resume(WithServerResumeSessionDuration(30 * time.Second)).
		Fragment(65535).
		Acceptor(func(setup SetupPayload, sendingSocket CloseableRSocket) RSocket {
			// Handle close.
			sendingSocket.OnClose(func() {
				log.Println("sending socket is closed")
			})
			// Request to client.
			sendingSocket.RequestResponse(NewString("Ping", time.Now().String())).
				DoOnSuccess(func(ctx context.Context, s Subscription, elem Payload) {
					log.Println("response of Ping from client:", elem)
				}).
				SubscribeOn(ElasticScheduler()).
				Subscribe(context.Background())
			// Return responser which just echo.
			return NewAbstractSocket(
				FireAndForget(func(msg Payload) {
					log.Println("receive fnf:", msg)
				}),
				RequestResponse(func(msg Payload) Mono {
					return JustMono(msg)
				}),
				RequestStream(func(msg Payload) Flux {
					return Range(0, 3).Map(func(n int) Payload {
						return NewString(msg.DataUTF8(), fmt.Sprintf("This is response #%04d", n))
					})
				}),
				RequestChannel(func(msgs Publisher) Flux {
					return ToFlux(msgs)
				}),
			)
		}).
		Transport("tcp://0.0.0.0:7878").
		Serve(context.Background())
	panic(err)
}

func ExampleConnect() {
	cli, err := Connect().
		Resume().
		Fragment(65535).
		SetupPayload(NewString("Hello", "World")).
		Acceptor(func(socket RSocket) RSocket {
			return NewAbstractSocket(RequestResponse(func(msg Payload) Mono {
				return JustMono(NewString("Pong", time.Now().String()))
			}))
		}).
		Transport("tcp://127.0.0.1:7878").
		Start(context.Background())
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = cli.Close()
	}()
	// Simple FireAndForget.
	cli.FireAndForget(NewString("This is a FNF message.", ""))
	// Simple RequestResponse.
	cli.RequestResponse(NewString("This is a RequestResponse message.", "")).
		DoOnSuccess(func(ctx context.Context, s Subscription, elem Payload) {
			log.Println("response:", elem)
		}).
		Subscribe(context.Background())
	// RequestStream with backpressure. (one by one)
	cli.RequestStream(NewString("This is a RequestStream message.", "")).
		DoOnNext(func(ctx context.Context, s Subscription, elem Payload) {
			log.Println("next element in stream:", elem)
			s.Request(1)
		}).
		DoOnSubscribe(func(ctx context.Context, s Subscription) {
			s.Request(1)
		}).
		Subscribe(context.Background())
	// Simple RequestChannel.
	sendFlux := Range(0, 3).
		Map(func(n int) Payload {
			return NewString(fmt.Sprintf("This is a RequestChannel message #%d.", n), "")
		})
	cli.RequestChannel(sendFlux).
		DoOnNext(func(ctx context.Context, s Subscription, elem Payload) {
			log.Println("next element in channel:", elem)
		}).
		Subscribe(context.Background())
}

func TestAllInOne(t *testing.T) {
	addresses := map[string]string{
		"unix":      "unix:///tmp/rsocket.test.sock",
		"tcp":       "tcp://localhost:7878",
		"websocket": "ws://localhost:8080/test",
	}
	for k, v := range addresses {
		t.Run(k, func(t *testing.T) {
			execTesting(v, t)
		})
	}
}

func execTesting(addr string, t *testing.T) {
	const setupData, setupMetadata = "你好", "世界"
	const streamElements = int32(10)
	const channelElements = int32(10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func(ctx context.Context) {
		err := Receive().
			Fragment(128).
			Acceptor(func(setup SetupPayload, sendingSocket CloseableRSocket) RSocket {
				require.Equal(t, setupData, setup.DataUTF8(), "bad setup data")
				m, _ := setup.MetadataUTF8()
				require.Equal(t, setupMetadata, m, "bad setup metadata")
				return NewAbstractSocket(RequestResponse(func(msg Payload) Mono {
					require.Equal(t, "ping", msg.DataUTF8(), "bad ping data")
					return NewMono(func(ctx context.Context, sink MonoProducer) {
						m, _ := msg.MetadataUTF8()
						_ = sink.Success(NewString("pong", m))
					})
				}), RequestStream(func(msg Payload) Flux {
					d := msg.DataUTF8()
					return Range(0, int(streamElements)).Map(func(n int) Payload {
						return NewString(d, fmt.Sprintf("%d", n))
					})
				}), RequestChannel(func(msgs Publisher) Flux {
					return ToFlux(msgs)
				}))
			}).
			Transport(addr).
			Serve(ctx)
		if err != nil {
			log.Printf("---->%+v\n", err)
		}
		require.NoError(t, err, "serve failed")
	}(ctx)

	time.Sleep(500 * time.Millisecond)

	cli, err := Connect().
		Fragment(192).
		SetupPayload(NewString(setupData, setupMetadata)).
		Transport(addr).
		Start(context.Background())
	require.NoError(t, err, "connect failed")
	defer func() {
		_ = cli.Close()
	}()

	// RequestResponse
	longText := strings.Repeat("foobar", 100)
	req := NewString("ping", longText)
	cli.RequestResponse(req).
		DoOnSuccess(func(ctx context.Context, s Subscription, elem Payload) {
			require.Equal(t, "pong", elem.DataUTF8(), "bad pong")
			expect, _ := req.MetadataUTF8()
			actual, _ := elem.MetadataUTF8()
			require.Equal(t, expect, actual, "bad metadata")
		}).
		Subscribe(ctx)

	// RequestStream: one by one
	seq := int32(0)
	cli.RequestStream(NewString(longText, "")).
		DoOnNext(func(ctx context.Context, s Subscription, elem Payload) {
			m, _ := elem.MetadataUTF8()
			require.Equal(t, fmt.Sprintf("%d", atomic.LoadInt32(&seq)), m, "bad stream metadata")
			require.Equal(t, longText, elem.DataUTF8(), "bad stream data")
			atomic.AddInt32(&seq, 1)
			s.Request(1)
		}).
		DoOnSubscribe(func(ctx context.Context, s Subscription) {
			s.Request(1)
		}).
		Subscribe(ctx)
	require.Equal(t, streamElements, seq)

	// RequestChannel
	atomic.StoreInt32(&seq, 0)
	send := Range(0, int(channelElements)).
		Map(func(n int) Payload {
			return NewString(longText, fmt.Sprintf("%d", n))
		})
	cli.RequestChannel(send).
		DoOnNext(func(ctx context.Context, s Subscription, elem Payload) {
			m, _ := elem.MetadataUTF8()
			require.Equal(t, fmt.Sprintf("%d", atomic.LoadInt32(&seq)), m, "bad stream metadata")
			require.Equal(t, longText, elem.DataUTF8(), "bad stream data")
			atomic.AddInt32(&seq, 1)
		}).
		Subscribe(ctx)

}
