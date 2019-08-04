package rsocket_test

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jjeffcaii/reactor-go/scheduler"
	. "github.com/rsocket/rsocket-go"
	. "github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/rsocket/rsocket-go/rx/mono"
	"github.com/stretchr/testify/require"
)

const (
	setupData, setupMetadata = "你好", "世界"
	streamElements           = int32(3)
	channelElements          = int32(3)
)

var testData = "Hello World!"

func TestAllInOne(t *testing.T) {
	//logger.SetLevel(logger.LevelDebug)
	addresses := map[string]string{
		"unix":      "unix:///tmp/rsocket.test.sock",
		"tcp":       "tcp://localhost:7878",
		"websocket": "ws://localhost:8080/test",
	}
	for k, v := range addresses {
		testAll(k, v, t)
	}
}

func testRequestResponse(ctx context.Context, cli Client, t *testing.T) {
	req := NewString("ping", testData)
	elem, err := cli.RequestResponse(req).Block(ctx)
	require.NoError(t, err, "call RequestResponse failed")
	require.Equal(t, "pong", elem.DataUTF8(), "bad pong")
	expect, _ := req.MetadataUTF8()
	actual, _ := elem.MetadataUTF8()
	require.Equal(t, expect, actual, "bad metadata")
}

func testRequestStream(ctx context.Context, cli Client, t *testing.T) {
	done := make(chan struct{})
	seq := int32(0)
	_, err := cli.RequestStream(NewString(testData, "")).
		DoFinally(func(s rx.SignalType) {
			close(done)
		}).
		DoOnNext(func(elem Payload) {
			m, _ := elem.MetadataUTF8()
			require.Equal(t, fmt.Sprintf("%d", atomic.LoadInt32(&seq)), m, "bad stream metadata")
			require.Equal(t, testData, elem.DataUTF8(), "bad stream data")
			atomic.AddInt32(&seq, 1)
		}).
		BlockLast(ctx)
	<-done
	require.NoError(t, err, "block last failed")
	require.Equal(t, streamElements, seq)
}

func testRequestStreamOneByOne(ctx context.Context, cli Client, t *testing.T) {
	done := make(chan struct{})
	var su rx.Subscription
	seq := int32(0)
	cli.RequestStream(NewString(testData, "")).
		DoFinally(func(s rx.SignalType) {
			close(done)
		}).
		DoOnNext(func(elem Payload) {
			m, _ := elem.MetadataUTF8()
			require.Equal(t, fmt.Sprintf("%d", atomic.LoadInt32(&seq)), m, "bad stream metadata")
			require.Equal(t, testData, elem.DataUTF8(), "bad stream data")
			atomic.AddInt32(&seq, 1)
			su.Request(1)
		}).
		Subscribe(ctx, rx.OnSubscribe(func(s rx.Subscription) {
			su = s
			su.Request(1)
		}))
	<-done
	require.Equal(t, streamElements, seq)
}

func testRequestChannel(ctx context.Context, cli Client, t *testing.T) {
	// RequestChannel
	send := flux.Create(func(ctx context.Context, s flux.Sink) {
		for i := 0; i < int(channelElements); i++ {
			s.Next(NewString(testData, fmt.Sprintf("%d", i)))
		}
		s.Complete()
	})

	var seq int

	_, err := cli.RequestChannel(send).
		DoOnNext(func(elem Payload) {
			log.Println(elem)
			m, _ := elem.MetadataUTF8()
			require.Equal(t, fmt.Sprintf("%d", seq), m, "bad channel metadata")
			require.Equal(t, testData, elem.DataUTF8(), "bad channel data")
			seq++
		}).
		BlockLast(ctx)
	require.NoError(t, err, "block last failed")
}

func testRequestChannelOneByOne(ctx context.Context, cli Client, t *testing.T) {
	// RequestChannel
	send := flux.Create(func(ctx context.Context, s flux.Sink) {
		for i := 0; i < int(channelElements); i++ {
			s.Next(NewString(testData, fmt.Sprintf("%d", i)))
		}
		s.Complete()
	})

	var seq int

	done := make(chan struct{})

	var su rx.Subscription

	cli.RequestChannel(send).
		DoFinally(func(s rx.SignalType) {
			require.Equal(t, rx.SignalComplete, s, "bad signal type")
			close(done)
		}).
		DoOnNext(func(elem Payload) {
			log.Println(elem)
			m, _ := elem.MetadataUTF8()
			require.Equal(t, fmt.Sprintf("%d", seq), m, "bad channel metadata")
			require.Equal(t, testData, elem.DataUTF8(), "bad channel data")
			seq++
		}).
		Subscribe(ctx, rx.OnNext(func(elem Payload) {
			su.Request(1)
		}), rx.OnSubscribe(func(s rx.Subscription) {
			su = s
			su.Request(1)
		}))
	<-done
}

func testAll(proto string, addr string, t *testing.T) {
	//const channelElements = int32(10)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func(ctx context.Context) {
		err := Receive().
			//Fragment(128).
			Acceptor(func(setup SetupPayload, sendingSocket CloseableRSocket) RSocket {
				require.Equal(t, setupData, setup.DataUTF8(), "bad setup data")
				m, _ := setup.MetadataUTF8()
				require.Equal(t, setupMetadata, m, "bad setup metadata")
				return NewAbstractSocket(
					RequestResponse(func(msg Payload) mono.Mono {
						require.Equal(t, "ping", msg.DataUTF8(), "bad ping data")
						return mono.Create(func(ctx context.Context, sink mono.Sink) {
							m, _ := msg.MetadataUTF8()
							sink.Success(NewString("pong", m))
						})
					}),
					RequestStream(func(msg Payload) flux.Flux {
						d := msg.DataUTF8()
						return flux.Create(func(ctx context.Context, s flux.Sink) {
							for i := 0; i < int(streamElements); i++ {
								s.Next(NewString(d, fmt.Sprintf("%d", i)))
							}
							s.Complete()
						})
					}),
					RequestChannel(func(inputs rx.Publisher) flux.Flux {
						var foo int32
						inputs.(flux.Flux).
							DoFinally(func(s rx.SignalType) {
								require.Equal(t, int(channelElements), int(atomic.LoadInt32(&foo)), "bad amount")
							}).
							SubscribeOn(scheduler.Elastic()).
							Subscribe(context.Background(), rx.OnNext(func(input Payload) {
								atomic.AddInt32(&foo, 1)
							}))

						return flux.Create(func(ctx context.Context, s flux.Sink) {
							for i := 0; i < int(channelElements); i++ {
								s.Next(NewString(testData, fmt.Sprintf("%d", i)))
							}
							s.Complete()
						})
					}),
				)
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

	t.Run(fmt.Sprintf("RequestResponse_%s", proto), func(t *testing.T) {
		testRequestResponse(ctx, cli, t)
	})
	t.Run(fmt.Sprintf("RequestStream_%s", proto), func(t *testing.T) {
		testRequestStream(ctx, cli, t)
	})
	t.Run(fmt.Sprintf("RequestStreamOneByOne_%s", proto), func(t *testing.T) {
		testRequestStreamOneByOne(ctx, cli, t)
	})
	t.Run(fmt.Sprintf("RequestChannel_%s", proto), func(t *testing.T) {
		testRequestChannel(ctx, cli, t)
	})
	t.Run(fmt.Sprintf("RequestChannelOneByOne_%s", proto), func(t *testing.T) {
		testRequestChannelOneByOne(ctx, cli, t)
	})

}
