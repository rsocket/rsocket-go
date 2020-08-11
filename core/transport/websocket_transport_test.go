package transport_test

import (
	"bytes"
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/stretchr/testify/assert"
)

func TestNewWebsocketServerTransport(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	listener := newMockNetListener(ctrl)

	fakeConnChan := make(chan net.Conn, 1)

	conn := newMockNetConn(ctrl)

	conn.EXPECT().
		RemoteAddr().
		DoAndReturn(func() net.Addr {
			addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:8080")
			assert.NoError(t, err, "bad addr")
			return addr
		}).
		AnyTimes()
	conn.EXPECT().
		LocalAddr().
		DoAndReturn(func() net.Addr {
			addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:18080")
			assert.NoError(t, err, "bad addr")
			return addr
		}).
		AnyTimes()
	conn.EXPECT().SetReadDeadline(gomock.Any()).Return(nil).AnyTimes()
	conn.EXPECT().Close().Return(nil).Times(1)
	bf := &bytes.Buffer{}
	conn.EXPECT().
		Read(gomock.Any()).
		DoAndReturn(func(b []byte) (int, error) {
			return bf.Read(b)
		}).
		AnyTimes()
	conn.EXPECT().
		Write(gomock.Any()).
		DoAndReturn(func(b []byte) (int, error) {
			return bf.Write(b)
		}).
		AnyTimes()

	fakeConnChan <- conn

	listener.EXPECT().
		Accept().
		DoAndReturn(func() (net.Conn, error) {
			c, ok := <-fakeConnChan
			if !ok {
				return nil, io.EOF
			}
			return c, nil
		}).
		AnyTimes()
	listener.EXPECT().Close().AnyTimes()

	tp := transport.NewWebsocketServerTransport(func() (net.Listener, error) {
		return listener, nil
	}, "")

	notifier := make(chan struct{})

	done := make(chan struct{})

	go func() {
		defer close(done)
		err := tp.Listen(context.Background(), notifier)
		assert.True(t, err == nil || err == io.EOF)
	}()

	_, ok := <-notifier
	assert.True(t, ok, "notifier should return ok=true")

	time.Sleep(100 * time.Millisecond)

	close(fakeConnChan)

	<-done
}

func TestNewWebsocketServerTransport_Broken(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tp := transport.NewWebsocketServerTransport(func() (net.Listener, error) {
		return nil, fakeErr
	}, "")
	tp.Accept(func(ctx context.Context, tp *transport.Transport, onClose func(*transport.Transport)) {
	})

	notifier := make(chan struct{})

	done := make(chan struct{})

	go func() {
		defer close(done)
		err := tp.Listen(context.Background(), notifier)
		assert.Equal(t, fakeErr, errors.Cause(err))
	}()

	_, ok := <-notifier
	assert.False(t, ok, "notifier should return ok=false")

	<-done
}
