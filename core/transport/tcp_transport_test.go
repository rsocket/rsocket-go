package transport_test

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/stretchr/testify/assert"
)

func InitTcpServerTransport(t *testing.T) (*gomock.Controller, *mockNetListener, transport.ServerTransport) {
	ctrl := gomock.NewController(t)
	listener := newMockNetListener(ctrl)
	tp := transport.NewTCPServerTransport(func(ctx context.Context) (net.Listener, error) {
		return listener, nil
	})
	return ctrl, listener, tp
}

func TestTcpServerTransport_ListenBroken(t *testing.T) {
	tp := transport.NewTCPServerTransport(func(ctx context.Context) (net.Listener, error) {
		return nil, fakeErr
	})

	defer tp.Close()

	done := make(chan struct{})

	notifier := make(chan bool)
	go func() {
		defer close(done)
		err := tp.Listen(context.Background(), notifier)
		assert.Equal(t, fakeErr, errors.Cause(err), "should caused by fake error")
	}()
	ok := <-notifier
	assert.False(t, ok)

	<-done
}

func TestTcpServerTransport_Listen(t *testing.T) {
	ctrl, listener, tp := InitTcpServerTransport(t)
	defer ctrl.Finish()

	listener.EXPECT().Accept().Return(nil, io.EOF).AnyTimes()
	listener.EXPECT().Close().Times(1)

	done := make(chan struct{})

	ctx, cancel := context.WithCancel(context.Background())
	notifier := make(chan bool)
	go func() {
		defer close(done)
		err := tp.Listen(ctx, notifier)
		assert.True(t, err == nil || err == io.EOF)
	}()
	ok := <-notifier
	assert.True(t, ok)

	time.Sleep(100 * time.Millisecond)
	cancel()

	<-done
}

func TestTcpServerTransport_Accept(t *testing.T) {
	ctrl, listener, tp := InitTcpServerTransport(t)
	defer ctrl.Finish()
	defer tp.Close()

	connChan := make(chan net.Conn, 1)
	listener.EXPECT().
		Accept().
		DoAndReturn(func() (net.Conn, error) {
			c, ok := <-connChan
			if !ok {
				return nil, io.EOF
			}
			return c, nil
		}).
		AnyTimes()
	listener.EXPECT().Close().Times(1)

	tp.Accept(func(ctx context.Context, tp *transport.Transport, onClose func(*transport.Transport)) {
		defer onClose(tp)
		err := tp.Start(ctx)
		assert.True(t, err == nil || err == io.EOF)
	})

	done := make(chan struct{})

	notifier := make(chan bool)
	go func() {
		defer close(done)
		err := tp.Listen(context.Background(), notifier)
		assert.True(t, err == nil || err == io.EOF)
	}()

	ok := <-notifier
	assert.True(t, ok, "notifier failed")

	c := newMockNetConn(ctrl)

	c.EXPECT().Read(gomock.Any()).Return(0, io.EOF).AnyTimes()
	c.EXPECT().Close().Times(1)

	connChan <- c

	time.Sleep(100 * time.Millisecond)
	close(connChan)

	<-done
}

func TestTcpServerTransport_AcceptBroken(t *testing.T) {
	ctrl, listener, tp := InitTcpServerTransport(t)
	defer ctrl.Finish()

	listener.EXPECT().
		Accept().
		Return(nil, fakeErr).
		AnyTimes()
	listener.EXPECT().Close().Times(1)

	tp.Accept(func(ctx context.Context, tp *transport.Transport, onClose func(*transport.Transport)) {
		defer onClose(tp)
		err := tp.Start(ctx)
		assert.True(t, err == nil || err == io.EOF)
	})

	done := make(chan struct{})

	notifier := make(chan bool)
	go func() {
		defer close(done)
		err := tp.Listen(context.Background(), notifier)
		assert.Error(t, err, "should be error")
		assert.Equal(t, fakeErr, errors.Cause(err), "should caused by fake error")
	}()

	ok := <-notifier
	assert.True(t, ok, "notifier failed")

	<-done
}

func TestNewTcpServerTransportWithAddr(t *testing.T) {
	assert.NotPanics(t, func() {
		tp := transport.NewTCPServerTransportWithAddr("tcp", ":9999", nil)
		assert.NotNil(t, tp)
	})
	assert.NotPanics(t, func() {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}
		tp := transport.NewTCPServerTransportWithAddr("tcp", ":9999", tlsConfig)
		assert.NotNil(t, tp)
	})
}
