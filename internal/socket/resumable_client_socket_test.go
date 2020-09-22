package socket_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/internal/fragmentation"
	"github.com/rsocket/rsocket-go/internal/socket"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

var fakeResumableSetup = &socket.SetupInfo{
	Version:           core.DefaultVersion,
	MetadataMimeType:  fakeMimeType,
	DataMimeType:      fakeMimeType,
	Metadata:          fakeMetadata,
	Data:              fakeData,
	KeepaliveLifetime: 90 * time.Second,
	KeepaliveInterval: 30 * time.Second,
	Token:             fakeToken,
}

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

	onCloseCalled := atomic.NewBool(false)
	rcs.OnClose(func(err error) {
		onCloseCalled.CAS(false, true)
	})

	defer func() {
		err := rcs.Close()
		assert.NoError(t, err)
		time.Sleep(500 * time.Millisecond)
		assert.True(t, onCloseCalled.Load())
	}()

	err := rcs.Setup(context.Background(), fakeResumableSetup)
	assert.NoError(t, err)

	requestId := atomic.NewUint32(1)
	nextRequestId := func() uint32 {
		return requestId.Add(2) - 2
	}

	result, err := rcs.RequestResponse(payload.New(fakeData, fakeMetadata)).
		DoOnSubscribe(func(ctx context.Context, s rx.Subscription) {
			readChan <- framing.NewPayloadFrame(nextRequestId(), fakeData, fakeMetadata, core.FlagComplete)
		}).
		Block(context.Background())
	assert.NoError(t, err, "request response failed")
	assert.Equal(t, fakeData, result.Data(), "response data doesn't match")
	assert.Equal(t, fakeMetadata, extractMetadata(result), "response metadata doesn't match")

	var stream []payload.Payload
	_, err = rcs.RequestStream(payload.New(fakeData, fakeMetadata)).
		DoOnNext(func(input payload.Payload) error {
			stream = append(stream, input)
			return nil
		}).
		DoOnSubscribe(func(ctx context.Context, s rx.Subscription) {
			nextId := nextRequestId()
			readChan <- framing.NewPayloadFrame(nextId, fakeData, fakeMetadata, core.FlagNext)
			readChan <- framing.NewPayloadFrame(nextId, fakeData, fakeMetadata, core.FlagNext)
			readChan <- framing.NewPayloadFrame(nextId, fakeData, fakeMetadata, core.FlagNext|core.FlagComplete)
		}).
		BlockLast(context.Background())
	assert.NoError(t, err, "request stream failed")

	// When a fatal error occurred, client should be stopped immediately.
	fatalErr := []byte("fatal error")
	readChan <- framing.NewErrorFrame(0, core.ErrorCodeRejected, fatalErr)
	time.Sleep(100 * time.Millisecond)
	err = ds.GetError()
	assert.Error(t, err, "should get error")
	assert.Equal(t, fatalErr, err.(core.CustomError).ErrorData())
}

func TestResumeClientSocket_Setup_Broken(t *testing.T) {
	c := socket.NewClientDuplexConnection(fragmentation.MaxFragment, 90*time.Second)
	s := socket.NewResumableClientSocket(func(ctx context.Context) (*transport.Transport, error) {
		return nil, fakeErr
	}, c)
	defer s.Close()
	err := s.Setup(context.Background(), fakeResumableSetup)
	assert.Error(t, err)
}

func TestResumeClientSocket_Setup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ds := socket.NewClientDuplexConnection(fragmentation.MaxFragment, 90*time.Second)

	readChanChan := make(chan chan core.Frame, 64)

	createTimes := atomic.NewInt32(0)
	rcs := socket.NewResumableClientSocket(func(ctx context.Context) (*transport.Transport, error) {
		if createTimes.Inc() >= 3 {
			return nil, fakeErr
		}

		conn, tp := InitTransportWithController(ctrl)

		// For test
		readChan := make(chan core.Frame, 64)

		conn.EXPECT().Close().AnyTimes()
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

		readChanChan <- readChan

		return tp, nil
	}, ds)

	onCloseCalled := atomic.NewBool(false)
	rcs.OnClose(func(err error) {
		onCloseCalled.CAS(false, true)
	})

	defer func() {
		err := rcs.Close()
		assert.NoError(t, err)
		time.Sleep(500 * time.Millisecond)
		assert.True(t, onCloseCalled.Load())
	}()

	err := rcs.Setup(context.Background(), fakeResumableSetup)
	assert.NoError(t, err)

	time.Sleep(100 * time.Millisecond)

	readChan := <-readChanChan
	close(readChan)

	time.Sleep(100 * time.Millisecond)

	readChan = <-readChanChan
	readChan <- framing.NewResumeOKFrame(0)
	time.Sleep(100 * time.Millisecond)
	readChan <- framing.NewErrorFrame(0, core.ErrorCodeRejectedResume, []byte("fake reject error"))
	close(readChan)
	time.Sleep(100 * time.Millisecond)
}
