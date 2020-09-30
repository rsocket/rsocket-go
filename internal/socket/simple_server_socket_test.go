package socket_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/internal/fragmentation"
	"github.com/rsocket/rsocket-go/internal/socket"
	"github.com/stretchr/testify/assert"
)

var fakeResponder = rsocket.NewAbstractSocket()

func TestSimpleServerSocket_Start(t *testing.T) {
	ctrl, conn, tp := InitTransport(t)
	defer ctrl.Finish()

	// For test
	readChan := make(chan core.BufferedFrame, 64)
	setupFrame := framing.NewSetupFrame(
		core.DefaultVersion,
		30*time.Second,
		90*time.Second,
		nil,
		fakeMimeType,
		fakeMimeType,
		fakeData,
		fakeMetadata,
		false,
	)
	readChan <- setupFrame

	conn.EXPECT().Close().Times(1)
	conn.EXPECT().SetCounter(gomock.Any()).AnyTimes()
	conn.EXPECT().Write(gomock.Any()).Return(nil).AnyTimes()
	conn.EXPECT().Flush().AnyTimes()
	conn.EXPECT().Read().DoAndReturn(func() (core.BufferedFrame, error) {
		next, ok := <-readChan
		if !ok {
			return nil, io.EOF
		}
		return next, nil
	}).AnyTimes()
	conn.EXPECT().SetDeadline(gomock.Any()).AnyTimes()

	firstFrame, err := tp.ReadFirst(context.Background())
	assert.NoError(t, err, "read first frame failed")
	assert.Equal(t, setupFrame, firstFrame, "first should be setup frame")

	close(readChan)

	c := socket.NewServerDuplexConnection(fragmentation.MaxFragment, nil)
	ss := socket.NewSimpleServerSocket(c)
	ss.SetResponder(fakeResponder)
	ss.SetTransport(tp)

	assert.Equal(t, false, ss.Pause(), "should always returns false")
	token, ok := ss.Token()
	assert.False(t, ok)
	assert.Nil(t, token, "token should be nil")

	done := make(chan struct{})
	go func() {
		defer close(done)
		err := ss.Start(context.Background())
		assert.NoError(t, err, "start server socket failed")
	}()

	err = tp.Start(context.Background())
	assert.NoError(t, err, "start transport failed")

	_ = c.Close()

	<-done
}
