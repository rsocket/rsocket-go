package rsocket_test

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jjeffcaii/reactor-go"
	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/extension"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/lease"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/rsocket/rsocket-go/rx/mono"
)

const (
	setupData, setupMetadata = "你好", "世界"
	streamElements           = int32(3)
	channelElements          = int32(2)
)

var (
	fakeErr       = errors.New("fake error")
	fakeData      = "fake data"
	fakeMetadata  = "fake metadata"
	fakeToken     = []byte(time.Now().String())
	fakeRequest   = payload.NewString(fakeData, fakeMetadata)
	fakeResponser = NewAbstractSocket()
)

func TestResume(t *testing.T) {
	sessionTimeout := 2 * time.Second
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		time.Sleep(1 * time.Second)
		assert.Zero(t, common.CountBorrowed())
	}()

	started := make(chan struct{})

	connected := int32(0)

	defer func() {
		assert.Equal(t, int32(1), atomic.LoadInt32(&connected), "connected should be 1")
	}()

	go func(ctx context.Context) {
		_ = Receive().
			OnStart(func() {
				close(started)
			}).
			Resume(WithServerResumeSessionDuration(sessionTimeout)).
			Acceptor(func(ctx context.Context, setup payload.SetupPayload, sendingSocket CloseableRSocket) (responder RSocket, err error) {
				atomic.AddInt32(&connected, 1)
				responder = NewAbstractSocket(
					RequestResponse(func(msg payload.Payload) mono.Mono {
						return mono.Just(msg)
					}),
				)
				return
			}).
			Transport(TCPServer().SetAddr(":9797").Build()).
			Serve(ctx)
	}(ctx)

	<-started

	ch := make(chan net.Listener, 1)

	proxyPort := 7979
	proxyAddr := fmt.Sprintf(":%d", proxyPort)
	upstreamAddr := "127.0.0.1:9797"
	go startProxy(proxyAddr, ch, upstreamAddr)

	time.Sleep(200 * time.Millisecond)

	cli, err := Connect().
		Resume(WithClientResumeToken(func() []byte {
			return fakeToken
		})).
		Transport(TCPClient().SetHostAndPort("127.0.0.1", proxyPort).Build()).
		Start(ctx)
	assert.NoError(t, err, "connect failed")
	defer cli.Close()

	res, release, err := cli.RequestResponse(fakeRequest).BlockUnsafe(ctx)
	assert.NoError(t, err, "request failed")
	assert.True(t, payload.Equal(res, fakeRequest))
	release()

	// shutdown the proxy
	_ = (<-ch).Close()
	time.Sleep(100 * time.Millisecond)

	// restart the proxy
	go startProxy(proxyAddr, ch, upstreamAddr)

	// client should request correctly.
	res, release, err = cli.RequestResponse(fakeRequest).BlockUnsafe(ctx)
	assert.NoError(t, err, "request failed")
	assert.True(t, payload.Equal(res, fakeRequest))
	release()

	// shutdown the proxy again and sleep until server session expired
	_ = (<-ch).Close()
	time.Sleep(sessionTimeout + 100*time.Millisecond)

	// restart the proxy again
	go startProxy(proxyAddr, ch, upstreamAddr)

	defer func() {
		_ = (<-ch).Close()
	}()

	// client should request failed
	_, _, err = cli.RequestResponse(payload.NewString("vvv", "vvv")).BlockUnsafe(ctx)
	assert.Error(t, err, "should return error")
}

func TestReceiveWithBadArgs(t *testing.T) {
	err := Receive().
		Fragment(-999).
		Acceptor(func(ctx context.Context, setup payload.SetupPayload, sendingSocket CloseableRSocket) (RSocket, error) {
			return fakeResponser, nil
		}).
		Transport(TCPServer().SetHostAndPort("127.0.0.1", DefaultPort).Build()).
		Serve(context.Background())
	assert.Error(t, err, "should serve failed")

	err = Receive().
		Acceptor(func(ctx context.Context, setup payload.SetupPayload, sendingSocket CloseableRSocket) (RSocket, error) {
			return fakeResponser, nil
		}).
		Transport(func(ctx context.Context) (transport.ServerTransport, error) {
			return nil, fakeErr
		}).
		Serve(context.Background())
	assert.Error(t, err, "should serve failed")
}

func TestConnectWithBadArgs(t *testing.T) {
	_, err := Connect().
		Fragment(-999).
		Transport(TCPClient().SetHostAndPort("127.0.0.1", DefaultPort).Build()).
		Start(context.Background())
	assert.Error(t, err, "should connect failed")
}

func TestConnectBroken(t *testing.T) {
	started := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	port := 8787

	go func(ctx context.Context) {
		_ = Receive().
			OnStart(func() {
				close(started)
			}).
			Acceptor(func(ctx context.Context, setup payload.SetupPayload, sendingSocket CloseableRSocket) (RSocket, error) {
				return fakeResponser, nil
			}).
			Transport(TCPServer().SetAddr(fmt.Sprintf(":%d", port)).Build()).
			Serve(ctx)
	}(ctx)

	<-started

	time.Sleep(500 * time.Millisecond)

	wg := sync.WaitGroup{}
	wg.Add(2)

	go func() {
		defer wg.Done()
		cli, err := Connect().Resume().Transport(TCPClient().SetHostAndPort("127.0.0.1", port).Build()).Start(ctx)
		require.NoError(t, err, "connect failed")
		defer cli.Close()
		_, _, err = cli.RequestResponse(fakeRequest).BlockUnsafe(ctx)
		assert.Error(t, err, "should connect failed")
	}()

	go func() {
		defer wg.Done()
		cli, err := Connect().Lease().Transport(TCPClient().SetHostAndPort("127.0.0.1", port).Build()).Start(ctx)
		require.NoError(t, err, "connect failed")
		defer cli.Close()
		_, _, err = cli.RequestResponse(fakeRequest).BlockUnsafe(ctx)
		assert.Error(t, err, "should connect failed")
	}()
	wg.Wait()
}

func TestBiDirection(t *testing.T) {
	testPort := 7777

	started := make(chan struct{})

	res := make(chan payload.Payload)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func(ctx context.Context) {
		l, _ := lease.NewSimpleFactory(3*time.Second, 1*time.Second, 1*time.Second, 10)
		_ = Receive().
			Lease(l).
			Resume(WithServerResumeSessionDuration(1 * time.Minute)).
			Fragment(0).
			OnStart(func() {
				close(started)
			}).
			Acceptor(func(ctx context.Context, setup payload.SetupPayload, sendingSocket CloseableRSocket) (RSocket, error) {
				sendingSocket.MetadataPush(fakeRequest)
				sendingSocket.FireAndForget(fakeRequest)
				sendingSocket.RequestResponse(fakeRequest).
					DoOnSuccess(func(input payload.Payload) error {
						res <- payload.Clone(input)
						return nil
					}).
					Subscribe(context.Background())
				sendingSocket.MetadataPush(fakeRequest)
				return fakeResponser, nil
			}).
			Transport(TCPServer().SetHostAndPort("127.0.0.1", testPort).Build()).
			Serve(ctx)
	}(ctx)

	<-started

	onCloseCalled := int32(0)

	fireAndForget := make(chan payload.Payload, 1)
	metadataPush := make(chan payload.Payload, 1)

	client, err := Connect().
		OnClose(func(err error) {
			atomic.StoreInt32(&onCloseCalled, 1)
		}).
		Fragment(0).
		Lease().
		KeepAlive(1*time.Minute, 10*time.Second, 3).
		DataMimeType(extension.TextPlain.String()).
		MetadataMimeType(extension.TextPlain.String()).
		Resume(WithClientResumeToken(func() []byte {
			return []byte("fake resume token")
		})).
		Acceptor(func(ctx context.Context, socket RSocket) RSocket {
			// echo anything
			return NewAbstractSocket(
				RequestResponse(func(msg payload.Payload) mono.Mono {
					return mono.Just(msg)
				}),
				MetadataPush(func(msg payload.Payload) {
					metadataPush <- payload.Clone(msg)
				}),
				FireAndForget(func(msg payload.Payload) {
					fireAndForget <- payload.Clone(msg)
				}),
			)
		}).
		Transport(TCPClient().SetHostAndPort("127.0.0.1", testPort).Build()).
		Start(ctx)
	assert.NoError(t, err, "connect failed")
	defer func() {
		assert.NoError(t, client.Close(), "should not return error")
		assert.Equal(t, int32(1), atomic.LoadInt32(&onCloseCalled), "onClose should be called")
	}()
	next := <-res
	assert.True(t, payload.Equal(fakeRequest, next), "request and response doesn't match")

	next = <-fireAndForget
	assert.True(t, payload.Equal(fakeRequest, next), "request and response doesn't match")

	next = <-metadataPush
	assert.Equal(t, getMetadata(fakeRequest), getMetadata(next))
}

func getMetadata(p payload.Payload) []byte {
	m, _ := p.Metadata()
	return m
}

func TestSuite(t *testing.T) {
	m := []string{
		"tcp",
		"websocket",
	}
	c := []transport.ClientTransporter{
		TCPClient().SetHostAndPort("127.0.0.1", 7878).Build(),
		WebsocketClient().SetURL("ws://127.0.0.1:8080/test").Build(),
	}
	s := []transport.ServerTransporter{
		TCPServer().SetAddr(":7878").Build(),
		WebsocketServer().SetAddr("127.0.0.1:8080").SetPath("/test").Build(),
	}

	for i := 0; i < len(m); i++ {
		testAll(t, m[i], c[i], s[i])
	}
}

type ctxKey string

func testAll(t *testing.T, proto string, clientTp transport.ClientTransporter, serverTp transport.ServerTransporter) {
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), ctxKey("foo"), "bar"))
	defer cancel()

	serving := make(chan struct{})

	go func(ctx context.Context) {
		err := Receive().
			Fragment(128).
			OnStart(func() {
				close(serving)
			}).
			Acceptor(func(ctx context.Context, setup payload.SetupPayload, sendingSocket CloseableRSocket) (RSocket, error) {
				assert.Equal(t, "bar", ctx.Value(ctxKey("foo")), "context value doesn't match")
				assert.Equal(t, setupData, setup.DataUTF8(), "bad setup data")
				m, _ := setup.MetadataUTF8()
				assert.Equal(t, setupMetadata, m, "bad setup metadata")
				return NewAbstractSocket(
					RequestResponse(func(msg payload.Payload) mono.Mono {
						assert.Equal(t, "ping", msg.DataUTF8(), "bad ping data")
						return mono.Create(func(ctx context.Context, sink mono.Sink) {
							assert.Equal(t, "bar", ctx.Value(ctxKey("foo")), "context value doesn't match")
							m, _ := msg.MetadataUTF8()
							sink.Success(payload.NewString("pong", m))
						})
					}),
					RequestStream(func(msg payload.Payload) flux.Flux {
						d := msg.DataUTF8()
						return flux.Create(func(ctx context.Context, s flux.Sink) {
							assert.Equal(t, "bar", ctx.Value(ctxKey("foo")), "context value doesn't match")
							for i := 0; i < int(streamElements); i++ {
								s.Next(payload.NewString(d, fmt.Sprintf("%d", i)))
							}
							s.Complete()
						})
					}),
					RequestChannel(func(initialRequest payload.Payload, inputs flux.Flux) flux.Flux {
						received := new(int32)
						inputs.
							DoOnNext(func(input payload.Payload) error {
								atomic.AddInt32(received, 1)
								return nil
							}).
							DoOnComplete(func() {
								assert.Equal(t, channelElements, atomic.LoadInt32(received))
							}).
							Subscribe(context.Background())

						return flux.Create(func(ctx context.Context, s flux.Sink) {
							assert.Equal(t, "bar", ctx.Value(ctxKey("foo")), "context value doesn't match")
							for i := 0; i < int(channelElements); i++ {
								s.Next(payload.NewString(fakeData, fmt.Sprintf("%d_from_server", i)))
							}
							s.Complete()
						})
					}),
				), nil
			}).
			Transport(serverTp).
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
		SetupPayload(payload.NewString(setupData, setupMetadata)).
		Transport(clientTp).
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
	req := payload.NewString("ping", fakeData)
	elem, release, err := cli.RequestResponse(req).BlockUnsafe(ctx)
	assert.NoError(t, err, "call RequestResponse failed")
	defer release()
	assert.Equal(t, "pong", elem.DataUTF8(), "bad pong")
	expect, _ := req.MetadataUTF8()
	actual, _ := elem.MetadataUTF8()
	assert.Equal(t, expect, actual, "bad metadata")
}

func testRequestStream(ctx context.Context, cli Client, t *testing.T) {
	done := make(chan struct{})
	seq := int32(0)
	_, err := cli.RequestStream(payload.NewString(fakeData, "")).
		DoFinally(func(s rx.SignalType) {
			close(done)
		}).
		DoOnNext(func(elem payload.Payload) error {
			m, _ := elem.MetadataUTF8()
			assert.Equal(t, fmt.Sprintf("%d", atomic.LoadInt32(&seq)), m, "bad stream metadata")
			assert.Equal(t, fakeData, elem.DataUTF8(), "bad stream data")
			atomic.AddInt32(&seq, 1)
			return nil
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
	cli.RequestStream(payload.NewString(fakeData, "")).
		DoFinally(func(s rx.SignalType) {
			close(done)
		}).
		DoOnNext(func(elem payload.Payload) error {
			m, _ := elem.MetadataUTF8()
			assert.Equal(t, fmt.Sprintf("%d", atomic.LoadInt32(&seq)), m, "bad stream metadata")
			assert.Equal(t, fakeData, elem.DataUTF8(), "bad stream data")
			atomic.AddInt32(&seq, 1)
			su.Request(1)
			return nil
		}).
		Subscribe(ctx, rx.OnSubscribe(func(ctx context.Context, s rx.Subscription) {
			su = s
			su.Request(1)
		}))
	<-done
	assert.Equal(t, streamElements, seq)
}

func testRequestChannel(ctx context.Context, cli Client, t *testing.T) {
	// RequestChannel
	initialPayload := payload.NewString("This is a RequestChannel initial message.", "")
	send := flux.Create(func(ctx context.Context, s flux.Sink) {
		for i := 0; i < int(channelElements); i++ {
			s.Next(payload.NewString(fakeData, fmt.Sprintf("%d", i)))
		}
		s.Complete()
	})

	var seq int

	_, err := cli.RequestChannel(initialPayload, send).
		DoOnNext(func(elem payload.Payload) error {
			//fmt.Println(elem)
			m, _ := elem.MetadataUTF8()
			assert.Equal(t, fmt.Sprintf("%d_from_server", seq), m, "bad channel metadata")
			assert.Equal(t, fakeData, elem.DataUTF8(), "bad channel data")
			seq++
			return nil
		}).
		BlockLast(ctx)
	assert.NoError(t, err, "block last failed")
}

func testRequestChannelOneByOne(ctx context.Context, cli Client, t *testing.T) {
	// RequestChannel
	initialPayload := payload.NewString("This is a RequestChannel initial message.", "")
	send := flux.Create(func(ctx context.Context, s flux.Sink) {
		for i := 0; i < int(channelElements); i++ {
			s.Next(payload.NewString(fakeData, fmt.Sprintf("%d", i)))
		}
		s.Complete()
	})

	var seq int

	done := make(chan struct{})

	var su rx.Subscription

	cli.RequestChannel(initialPayload, send).
		DoFinally(func(s rx.SignalType) {
			assert.Equal(t, rx.SignalComplete, s, "bad signal type")
			close(done)
		}).
		DoOnNext(func(next payload.Payload) error {
			m, _ := next.MetadataUTF8()
			assert.Equal(t, fmt.Sprintf("%d_from_server", seq), m, "bad channel metadata")
			assert.Equal(t, fakeData, next.DataUTF8(), "bad channel data")
			seq++
			return nil
		}).
		Subscribe(ctx, rx.OnNext(func(elem payload.Payload) error {
			su.Request(1)
			return nil
		}), rx.OnSubscribe(func(ctx context.Context, s rx.Subscription) {
			su = s
			su.Request(1)
		}))
	<-done
}

// Starting a tcp proxy to simulate network broken
func startProxy(addr string, ch chan net.Listener, upstreamAddr string) {
	var (
		conns     []net.Conn
		upstreams []net.Conn
	)
	defer func() {
		for _, conn := range conns {
			_ = conn.Close()
		}
		for _, upstream := range upstreams {
			_ = upstream.Close()
		}
		log.Println("PROXY KILLED")
	}()
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return
	}
	ch <- l
	for {
		c, err := l.Accept()
		if err != nil {
			break
		}
		conns = append(conns, c)
		go func() {
			upstream, _ := net.Dial("tcp", upstreamAddr)
			upstreams = append(upstreams, upstream)
			wg := sync.WaitGroup{}
			wg.Add(2)

			go func() {
				defer wg.Done()
				_, _ = io.Copy(c, upstream)
			}()
			go func() {
				defer wg.Done()
				_, _ = io.Copy(upstream, c)
			}()
			wg.Wait()
		}()
	}

}

type delayedRSocket struct {
}

func (d delayedRSocket) FireAndForget(message payload.Payload) {
	panic("implement me")
}

func (d delayedRSocket) MetadataPush(message payload.Payload) {
	panic("implement me")
}

func (d delayedRSocket) RequestResponse(message payload.Payload) mono.Mono {
	return mono.Create(func(ctx context.Context, sink mono.Sink) {
		time.AfterFunc(300*time.Millisecond, func() {
			sink.Success(message)
		})
	})
}

func (d delayedRSocket) RequestStream(message payload.Payload) flux.Flux {
	panic("implement me")
}

func (d delayedRSocket) RequestChannel(initialRequest payload.Payload, messages flux.Flux) flux.Flux {
	panic("implement me")
}

func TestContextTimeout(t *testing.T) {
	var responder delayedRSocket
	started := make(chan struct{})
	go func() {
		_ = Receive().
			OnStart(func() {
				close(started)
			}).
			Acceptor(func(ctx context.Context, setup payload.SetupPayload, sendingSocket CloseableRSocket) (RSocket, error) {
				return responder, nil
			}).
			Transport(TCPServer().SetAddr(":8088").Build()).
			Serve(context.Background())
	}()

	<-started

	tp := TCPClient().SetAddr("127.0.0.1:8088").Build()

	// simulate timeout
	_, err := Connect().ConnectTimeout(1 * time.Nanosecond).Transport(tp).Start(context.Background())
	assert.Error(t, err, "should connect timeout")

	connected := make(chan bool)
	cli, err := Connect().
		OnConnect(func(c Client, err error) {
			connected <- err == nil
		}).
		ConnectTimeout(100 * time.Millisecond).
		Transport(tp).Start(context.Background())
	assert.NoError(t, err, "should connect success")
	defer cli.Close()
	assert.True(t, true, <-connected, "connected should be true")
	time.Sleep(200 * time.Millisecond)

	ctx, cancel2 := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel2()

	_, _, err = cli.RequestResponse(fakeRequest).BlockUnsafe(ctx)
	assert.Error(t, err, "should return error")
	assert.True(t, reactor.IsCancelledError(err), "should be cancelled error")

	_, _, err = cli.RequestResponse(fakeRequest).Timeout(100 * time.Millisecond).BlockUnsafe(context.Background())
	assert.Error(t, err, "should return error")
	assert.True(t, reactor.IsCancelledError(err), "should be cancelled error")

	_, release, err := cli.RequestResponse(fakeRequest).Timeout(400 * time.Millisecond).BlockUnsafe(context.Background())
	assert.NoError(t, err)
	release()
}

func TestEchoParallel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	started := make(chan struct{})

	go func() {
		_ = Receive().
			OnStart(func() {
				close(started)
			}).
			Acceptor(func(ctx context.Context, setup payload.SetupPayload, sendingSocket CloseableRSocket) (RSocket, error) {
				return NewAbstractSocket(RequestResponse(func(request payload.Payload) (response mono.Mono) {
					response = mono.Just(request)
					return
				})), nil
			}).
			Transport(TCPServer().SetHostAndPort("127.0.0.1", 7878).Build()).
			Serve(ctx)
	}()

	<-started

	cli, err := Connect().Transport(TCPClient().SetHostAndPort("127.0.0.1", 7878).Build()).Start(ctx)
	assert.NoError(t, err)
	defer cli.Close()

	const total = 1

	wg := new(sync.WaitGroup)
	wg.Add(total)

	req := payload.NewString(common.RandAlphanumeric(30), common.RandAlphanumeric(30))

	for i := 0; i < total; i++ {
		cli.RequestResponse(req).
			DoFinally(func(s rx.SignalType) {
				wg.Done()
			}).
			SubscribeOn(scheduler.Parallel()).
			Subscribe(ctx, rx.OnNext(func(res payload.Payload) error {
				assert.True(t, payload.Equal(req, res), "should be same payload")
				return nil
			}))
	}
	wg.Wait()
}
