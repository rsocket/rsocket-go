package socket_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/internal/fragmentation"
	"github.com/rsocket/rsocket-go/internal/socket"
	"github.com/stretchr/testify/assert"
)

func TestNewResumableClientSocket(t *testing.T) {
	ctrl, conn, tp := InitTransport(t)
	defer ctrl.Finish()

	// For test
	readChan := make(chan core.Frame, 64)

	conn.EXPECT().Close().Times(1)
	conn.EXPECT().SetCounter(gomock.Any()).Times(1)
	conn.EXPECT().Write(gomock.Any()).Return(nil).AnyTimes()
	conn.EXPECT().Flush().AnyTimes()
	conn.EXPECT().Read().DoAndReturn(func() (core.Frame, error) {
		next, ok := <-readChan
		if !ok {
			return nil, io.EOF
		}
		return next, nil
	}).AnyTimes()
	conn.EXPECT().SetDeadline(gomock.Any()).AnyTimes()

	ds := socket.NewClientDuplexConnection(fragmentation.MaxFragment, 90*time.Second)

	rcs := socket.NewResumableClientSocket(func(ctx context.Context) (*transport.Transport, error) {
		return tp, nil
	}, ds)

	defer rcs.Close()

	err := rcs.Setup(context.Background(), fakeSetup)
	assert.NoError(t, err)
}
