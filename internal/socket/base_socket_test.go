package socket_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/rsocket/rsocket-go/internal/fragmentation"
	"github.com/rsocket/rsocket-go/internal/socket"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

func TestBaseSocket(t *testing.T) {
	ctrl, conn, tp := InitTransport(t)
	defer ctrl.Finish()

	conn.EXPECT().Close().Times(1)
	conn.EXPECT().Write(gomock.Any()).Return(nil).AnyTimes()
	conn.EXPECT().Flush().AnyTimes()
	conn.EXPECT().Read().Return(nil, io.EOF).AnyTimes()
	conn.EXPECT().SetDeadline(gomock.Any()).AnyTimes()

	duplex := socket.NewClientDuplexConnection(context.Background(), nil, nil, fragmentation.MaxFragment, 90*time.Second)
	duplex.SetTransport(tp)

	go func() {
		_ = duplex.LoopWrite(context.Background())
	}()

	s := socket.NewBaseSocket(duplex)

	onClosedCalled := atomic.NewBool(false)

	s.OnClose(func(err error) {
		onClosedCalled.CAS(false, true)
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = tp.Start(context.Background())
	}()

	assert.NotPanics(t, func() {
		s.MetadataPush(fakeRequest)
		s.FireAndForget(fakeRequest)
		s.RequestResponse(fakeRequest)
		s.RequestStream(fakeRequest)
		s.RequestChannel(fakeRequest, flux.Just(fakeRequest))
	})

	<-done

	_ = s.Close()
	assert.Equal(t, true, onClosedCalled.Load())
}
