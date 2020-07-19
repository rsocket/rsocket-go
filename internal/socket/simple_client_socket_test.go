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

func TestNewClientWithBrokenTransporter(t *testing.T) {
	ds := socket.NewClientDuplexConnection(fragmentation.MaxFragment, 90*time.Second)
	// Must failed transporter
	transporter := func(ctx context.Context) (*transport.Transport, error) {
		return nil, fakeErr
	}
	cli := socket.NewClient(transporter, ds)
	err := cli.Setup(context.Background(), fakeSetup)
	assert.Equal(t, fakeErr, err, "should be fake error")
}

func TestNewClient(t *testing.T) {
	ctrl, conn, tp := Init(t)
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
	cli := socket.NewClient(func(ctx context.Context) (*transport.Transport, error) {
		return tp, nil
	}, ds)

	defer func() {
		err := cli.Close()
		assert.NoError(t, err, "close client failed")
	}()

	err := cli.Setup(context.Background(), fakeSetup)
	assert.NoError(t, err, "setup client failed")

	requestId := atomic.NewUint32(1)
	nextRequestId := func() uint32 {
		return requestId.Add(2) - 2
	}

	result, err := cli.RequestResponse(payload.New(fakeData, fakeMetadata)).
		DoOnSubscribe(func(s rx.Subscription) {
			readChan <- framing.NewPayloadFrame(nextRequestId(), fakeData, fakeMetadata, core.FlagComplete)
		}).
		Block(context.Background())
	assert.NoError(t, err, "request response failed")
	assert.Equal(t, fakeData, result.Data(), "response data doesn't match")
	assert.Equal(t, fakeMetadata, extractMetadata(result), "response metadata doesn't match")

	var stream []payload.Payload
	_, err = cli.RequestStream(payload.New(fakeData, fakeMetadata)).
		DoOnNext(func(input payload.Payload) {
			stream = append(stream, input)
		}).
		DoOnSubscribe(func(s rx.Subscription) {
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

func TestLease(t *testing.T) {
	ctrl, conn, tp := Init(t)
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
	cli := socket.NewClient(func(ctx context.Context) (*transport.Transport, error) {
		return tp, nil
	}, ds)

	defer func() {
		err := cli.Close()
		assert.NoError(t, err, "close client failed")
	}()

	setup := *fakeSetup
	setup.Lease = true
	err := cli.Setup(context.Background(), &setup)
	assert.NoError(t, err, "setup client failed")
	readChan <- framing.NewLeaseFrame(10*time.Second, 10, fakeMetadata)
	time.Sleep(3 * time.Second)
}

func extractMetadata(p payload.Payload) []byte {
	m, _ := p.Metadata()
	return m
}
