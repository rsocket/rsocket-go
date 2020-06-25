package rsocket_test

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	. "github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/logger"
	. "github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/rsocket/rsocket-go/rx/mono"
	"github.com/stretchr/testify/assert"
)

const (
	setupData, setupMetadata = "你好", "世界"
	streamElements           = int32(3)
	channelElements          = int32(2)
)

func init() {
	//logger.SetLevel(logger.LevelDebug)
	logger.SetFunc(logger.LevelInfo, func(s string, i ...interface{}) {
		fmt.Printf(s, i...)
	})
	logger.SetFunc(logger.LevelDebug, func(s string, i ...interface{}) {
		fmt.Printf(s, i...)
	})
	logger.SetFunc(logger.LevelWarn, func(s string, i ...interface{}) {
		fmt.Printf(s, i...)
	})
	logger.SetFunc(logger.LevelError, func(s string, i ...interface{}) {
		fmt.Printf(s, i...)
	})
}

var testData = "Hello World!"

func TestSuite(t *testing.T) {
	transports := map[string]Transporter{
		"tcp":       Tcp().Addr("127.0.0.1:7878").Build(),
		"websocket": Websocket().Url("ws://127.0.0.1:8080/test").Build(),
	}
	for k, v := range transports {
		testAll(t, k, v)
	}
}

func testAll(t *testing.T, proto string, tp Transporter) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serving := make(chan struct{})

	go func(ctx context.Context) {
		err := Receive().
			Fragment(128).
			OnStart(func() {
				close(serving)
			}).
			Acceptor(func(setup SetupPayload, sendingSocket CloseableRSocket) (RSocket, error) {
				assert.Equal(t, setupData, setup.DataUTF8(), "bad setup data")
				m, _ := setup.MetadataUTF8()
				assert.Equal(t, setupMetadata, m, "bad setup metadata")
				return NewAbstractSocket(
					RequestResponse(func(msg Payload) mono.Mono {
						assert.Equal(t, "ping", msg.DataUTF8(), "bad ping data")
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
						//var count int32
						//countPointer := &count
						receives := make(chan Payload)

						go func() {
							var count int32
							for range receives {
								count++
							}
							assert.Equal(t, channelElements, count, "bad channel amount")
						}()

						inputs.(flux.Flux).DoFinally(func(s rx.SignalType) {
							close(receives)
						}).Subscribe(context.Background(), rx.OnNext(func(input Payload) {
							//fmt.Println("rcv from channel:", input)
							receives <- input
						}))

						return flux.Create(func(ctx context.Context, s flux.Sink) {
							for i := 0; i < int(channelElements); i++ {
								s.Next(NewString(testData, fmt.Sprintf("%d_from_server", i)))
							}
							s.Complete()
						})
					}),
				), nil
			}).
			Transport(tp).
			Serve(ctx)
		fmt.Println("SERVER STOPPED!!!!!")
		if err != nil {
			fmt.Printf("---->%+v\n", err)
		}
		assert.NoError(t, err, "serve failed")
	}(ctx)

	<-serving

	cli, err := Connect().
		Fragment(192).
		SetupPayload(NewString(setupData, setupMetadata)).
		Transport(tp).
		Start(context.Background())
	assert.NoError(t, err, "connect failed")
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

func testRequestResponse(ctx context.Context, cli Client, t *testing.T) {
	req := NewString("ping", testData)
	elem, err := cli.RequestResponse(req).Block(ctx)
	assert.NoError(t, err, "call RequestResponse failed")
	assert.Equal(t, "pong", elem.DataUTF8(), "bad pong")
	expect, _ := req.MetadataUTF8()
	actual, _ := elem.MetadataUTF8()
	assert.Equal(t, expect, actual, "bad metadata")
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
			assert.Equal(t, fmt.Sprintf("%d", atomic.LoadInt32(&seq)), m, "bad stream metadata")
			assert.Equal(t, testData, elem.DataUTF8(), "bad stream data")
			atomic.AddInt32(&seq, 1)
		}).
		BlockLast(ctx)
	<-done
	assert.NoError(t, err, "block last failed")
	assert.Equal(t, streamElements, seq)
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
			assert.Equal(t, fmt.Sprintf("%d", atomic.LoadInt32(&seq)), m, "bad stream metadata")
			assert.Equal(t, testData, elem.DataUTF8(), "bad stream data")
			atomic.AddInt32(&seq, 1)
			su.Request(1)
		}).
		Subscribe(ctx, rx.OnSubscribe(func(s rx.Subscription) {
			su = s
			su.Request(1)
		}))
	<-done
	assert.Equal(t, streamElements, seq)
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
			//fmt.Println(elem)
			m, _ := elem.MetadataUTF8()
			assert.Equal(t, fmt.Sprintf("%d_from_server", seq), m, "bad channel metadata")
			assert.Equal(t, testData, elem.DataUTF8(), "bad channel data")
			seq++
		}).
		BlockLast(ctx)
	assert.NoError(t, err, "block last failed")
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
			assert.Equal(t, rx.SignalComplete, s, "bad signal type")
			close(done)
		}).
		DoOnNext(func(elem Payload) {
			fmt.Println(elem)
			m, _ := elem.MetadataUTF8()
			assert.Equal(t, fmt.Sprintf("%d_from_server", seq), m, "bad channel metadata")
			assert.Equal(t, testData, elem.DataUTF8(), "bad channel data")
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
