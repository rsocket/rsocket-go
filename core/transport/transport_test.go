package transport_test

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

func Init(t *testing.T) (*gomock.Controller, *MockConn, *transport.Transport) {
	ctrl := gomock.NewController(t)
	conn := NewMockConn(ctrl)
	tp := transport.NewTransport(conn)
	return ctrl, conn, tp
}

func TestTransport_Start(t *testing.T) {
	ctrl, conn, tp := Init(t)
	defer ctrl.Finish()

	conn.EXPECT().Close().Return(nil).Times(1)

	conn.EXPECT().Read().Return(nil, fakeErr).Times(1)
	err := tp.Start(context.Background())
	assert.Error(t, err, "should be an error")
	assert.True(t, errors.Cause(err) == fakeErr, "should be the fake error")

	conn.EXPECT().Read().Return(nil, io.EOF).Times(1)
	err = tp.Start(context.Background())
	assert.NoError(t, err, "there should be no error here if io.EOF occurred")
}

func TestTransport_RegisterHandler(t *testing.T) {
	ctrl, conn, tp := Init(t)
	defer ctrl.Finish()

	conn.EXPECT().Close().AnyTimes()
	conn.EXPECT().SetDeadline(gomock.Any()).AnyTimes()

	var cursor int
	fakeToken := []byte("fake-token")
	fakeDada := []byte("fake-data")
	fakeMetadata := []byte("fake-metadata")
	fakeMimeType := []byte("fake-mime-type")
	fakeFlag := core.FrameFlag(0)
	frames := []core.BufferedFrame{
		framing.NewSetupFrame(
			core.NewVersion(1, 0),
			30*time.Second,
			90*time.Second,
			nil,
			fakeMimeType,
			fakeMimeType,
			fakeDada,
			fakeMetadata,
			false),
		framing.NewMetadataPushFrame(fakeDada),
		framing.NewFireAndForgetFrame(1, fakeDada, fakeMetadata, fakeFlag),
		framing.NewRequestResponseFrame(1, fakeDada, fakeMetadata, fakeFlag),
		framing.NewRequestStreamFrame(1, 1, fakeDada, fakeMetadata, fakeFlag),
		framing.NewRequestChannelFrame(1, 1, fakeDada, fakeMetadata, fakeFlag),
		framing.NewRequestNFrame(1, 1, fakeFlag),
		framing.NewKeepaliveFrame(1, fakeDada, true),
		framing.NewCancelFrame(1),
		framing.NewErrorFrame(1, core.ErrorCodeApplicationError, fakeDada),
		framing.NewLeaseFrame(30*time.Second, 1, fakeMetadata),
		framing.NewPayloadFrame(1, fakeDada, fakeMetadata, fakeFlag),
		framing.NewResumeFrame(core.DefaultVersion, fakeToken, 1, 1),
		framing.NewResumeOKFrame(1),
		framing.NewErrorFrame(0, core.ErrorCodeRejected, fakeDada),
	}
	conn.EXPECT().
		Read().
		DoAndReturn(func() (core.BufferedFrame, error) {
			defer func() {
				cursor++
			}()
			if cursor >= len(frames) {
				return nil, io.EOF
			}
			return frames[cursor], nil
		}).
		AnyTimes()

	calls := make(map[core.FrameType]int)
	fakeHandler := func(frame core.BufferedFrame) (err error) {
		typ := frame.Header().Type()
		calls[typ] = calls[typ] + 1
		return nil
	}

	callsErrorWithZeroStreamID := atomic.NewInt32(0)

	tp.Handle(transport.OnSetup, fakeHandler)
	tp.Handle(transport.OnRequestResponse, fakeHandler)
	tp.Handle(transport.OnFireAndForget, fakeHandler)
	tp.Handle(transport.OnMetadataPush, fakeHandler)
	tp.Handle(transport.OnRequestStream, fakeHandler)
	tp.Handle(transport.OnRequestChannel, fakeHandler)
	tp.Handle(transport.OnKeepalive, fakeHandler)
	tp.Handle(transport.OnRequestN, fakeHandler)
	tp.Handle(transport.OnPayload, fakeHandler)
	tp.Handle(transport.OnError, fakeHandler)
	tp.Handle(transport.OnCancel, fakeHandler)
	tp.Handle(transport.OnResumeOK, fakeHandler)
	tp.Handle(transport.OnResume, fakeHandler)
	tp.Handle(transport.OnLease, fakeHandler)
	tp.Handle(transport.OnErrorWithZeroStreamID, func(frame core.BufferedFrame) (err error) {
		callsErrorWithZeroStreamID.Inc()
		return
	})

	toHaveBeenCalled := func(typ core.FrameType, n int) {
		called, ok := calls[typ]
		assert.True(t, ok, "%s have not been called", typ)
		assert.Equal(t, n, called, "%s have not been called %d times", n)
	}

	err := tp.Start(context.Background())
	assert.Error(t, err, "should be no error")

	for _, typ := range []core.FrameType{
		core.FrameTypeSetup,
		core.FrameTypeRequestResponse,
		core.FrameTypeRequestFNF,
		core.FrameTypeMetadataPush,
		core.FrameTypeRequestStream,
		core.FrameTypeRequestChannel,
		core.FrameTypeKeepalive,
		core.FrameTypeRequestN,
		core.FrameTypePayload,
		core.FrameTypeError,
		core.FrameTypeCancel,
		core.FrameTypeResume,
		core.FrameTypeResumeOK,
		core.FrameTypeLease,
	} {
		toHaveBeenCalled(typ, 1)
	}
	assert.Equal(t, int32(1), callsErrorWithZeroStreamID.Load(), "error frame with zero stream id has not been called")
}

func TestTransport_ReadFirst(t *testing.T) {
	ctrl, conn, tp := Init(t)
	defer ctrl.Finish()

	conn.EXPECT().Close().AnyTimes()

	conn.EXPECT().Read().Return(nil, fakeErr).Times(1)
	_, err := tp.ReadFirst(context.Background())
	assert.Error(t, err, "should be error")

	expect := framing.NewCancelFrame(1)
	conn.EXPECT().Read().Return(expect, nil).Times(1)
	actual, err := tp.ReadFirst(context.Background())
	assert.NoError(t, err, "should not be error")
	assert.Equal(t, expect, actual, "not match")
}

func TestTransport_Send(t *testing.T) {
	ctrl, conn, tp := Init(t)
	defer ctrl.Finish()

	conn.EXPECT().Write(gomock.Any()).Times(2)
	conn.EXPECT().Flush().Times(1)

	var err error

	err = tp.Send(framing.NewWriteableCancelFrame(1), false)
	assert.NoError(t, err, "send failed")

	err = tp.Send(framing.NewWriteableCancelFrame(1), true)
	assert.NoError(t, err, "send failed")
}

func TestTransport_Connection(t *testing.T) {
	ctrl, conn, tp := Init(t)
	defer ctrl.Finish()

	c := tp.Connection()
	assert.Equal(t, conn, c)
}

func TestTransport_Flush(t *testing.T) {
	ctrl, conn, tp := Init(t)
	defer ctrl.Finish()

	conn.EXPECT().Flush().Times(1)
	conn.EXPECT().SetCounter(gomock.Any()).Times(1)

	err := tp.Flush()
	assert.NoError(t, err, "flush failed")
	conn.SetCounter(core.NewTrafficCounter())
}

func TestTransport_Close(t *testing.T) {
	ctrl, conn, tp := Init(t)
	defer ctrl.Finish()

	conn.EXPECT().Close().Times(1)

	err := tp.Close()
	assert.NoError(t, err, "close transport failed")
}

func TestTransport_HandlerReturnsError(t *testing.T) {
	ctrl, conn, tp := Init(t)
	defer ctrl.Finish()

	conn.EXPECT().SetDeadline(gomock.Any()).AnyTimes()
	conn.EXPECT().Close().Times(1)
	conn.EXPECT().Read().Return(framing.NewCancelFrame(1), nil).MinTimes(1)

	tp.Handle(transport.OnCancel, func(_ core.BufferedFrame) error {
		return fakeErr
	})
	err := tp.Start(context.Background())
	assert.Equal(t, fakeErr, errors.Cause(err), "should caused by fakeError")
}

func TestTransport_EmptyHandler(t *testing.T) {
	ctrl, conn, tp := Init(t)
	defer ctrl.Finish()

	conn.EXPECT().SetDeadline(gomock.Any()).AnyTimes()
	conn.EXPECT().Close().Times(1)
	conn.EXPECT().Read().Return(framing.NewCancelFrame(1), nil).Times(1)

	err := tp.Start(context.Background())
	assert.True(t, transport.IsNoHandlerError(errors.Cause(err)), "should be no handler error")
}
