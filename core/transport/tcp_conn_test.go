package transport_test

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/logger"
	"github.com/stretchr/testify/assert"
)

func InitMockTcpConn(t *testing.T) (*gomock.Controller, *mockNetConn, *transport.TcpConn) {
	ctrl := gomock.NewController(t)
	nc := newMockNetConn(ctrl)
	tc := transport.NewTcpConn(nc)
	return ctrl, nc, tc
}

func TestTcpConn_Read_Empty(t *testing.T) {
	ctrl, nc, tc := InitMockTcpConn(t)
	defer ctrl.Finish()
	nc.EXPECT().Read(gomock.Any()).Return(0, fakeErr).AnyTimes()
	_, err := tc.Read()
	assert.Error(t, err, "should read failed")
}

func TestTcpConn_Read(t *testing.T) {
	ctrl, nc, tc := InitMockTcpConn(t)
	defer ctrl.Finish()

	bf := &bytes.Buffer{}
	c := core.NewCounter()
	tc.SetCounter(c)

	toBeWritten := []core.WriteableFrame{
		framing.NewWriteablePayloadFrame(1, fakeData, fakeMetadata, 0),
		framing.NewWriteableKeepaliveFrame(0, fakeData, true),
		framing.NewWriteableRequestResponseFrame(2, fakeData, fakeMetadata, 0),
	}

	var writtenBytes int

	for _, frame := range toBeWritten {
		n := frame.Len()
		if frame.Header().Resumable() {
			writtenBytes += n
		}
		_, _ = common.MustNewUint24(n).WriteTo(bf)
		_, _ = frame.WriteTo(bf)
	}

	nc.EXPECT().
		Read(gomock.Any()).
		DoAndReturn(func(b []byte) (int, error) {
			return bf.Read(b)
		}).
		AnyTimes()
	var results []core.Frame
	for {
		next, err := tc.Read()
		if err == io.EOF {
			break
		}
		assert.NoError(t, err, "read next frame failed")
		results = append(results, next)
	}
	assert.Equal(t, len(toBeWritten), len(results), "result amount does not match")
	for i := 0; i < len(results); i++ {
		assert.Equal(t, toBeWritten[i].Header(), results[i].Header(), "header does not match")
	}
	assert.Equal(t, writtenBytes, int(c.ReadBytes()), "read bytes doesn't match")
}

func TestTcpConn_SetDeadline(t *testing.T) {
	ctrl, nc, tc := InitMockTcpConn(t)
	defer ctrl.Finish()

	nc.EXPECT().SetReadDeadline(gomock.Any()).Times(1)
	err := tc.SetDeadline(time.Now())
	assert.NoError(t, err, "call setDeadline failed")
}

func TestTcpConn_Flush_Nothing(t *testing.T) {
	ctrl, nc, tc := InitMockTcpConn(t)
	defer ctrl.Finish()

	c := core.NewCounter()
	tc.SetCounter(c)

	nc.EXPECT().Write(gomock.Any()).Times(0)

	err := tc.Flush()
	assert.NoError(t, err, "flush failed")
	assert.Equal(t, 0, int(c.WriteBytes()), "bytes written should be zero")
}

func TestTcpConn_WriteWithBrokenConn(t *testing.T) {
	logger.SetLogger(nil)
	logger.SetLevel(logger.LevelDebug)
	ctrl, nc, tc := InitMockTcpConn(t)
	defer ctrl.Finish()
	nc.EXPECT().
		Write(gomock.Any()).
		Return(0, fakeErr).
		AnyTimes()
	_ = tc.Write(framing.NewWriteablePayloadFrame(1, fakeData, fakeMetadata, 0))
	err := tc.Flush()
	assert.Equal(t, fakeErr, errors.Cause(err), "should be fake error")
}

func TestTcpConn_WriteAndFlush(t *testing.T) {
	ctrl, nc, tc := InitMockTcpConn(t)
	defer ctrl.Finish()

	c := core.NewCounter()
	tc.SetCounter(c)

	nc.EXPECT().
		Write(gomock.Any()).
		DoAndReturn(func(b []byte) (int, error) {
			return len(b), nil
		}).
		Times(1)

	toBeWritten := []core.WriteableFrame{
		framing.NewWriteablePayloadFrame(1, fakeData, fakeMetadata, 0),
		framing.NewWriteableKeepaliveFrame(0, fakeData, true),
		framing.NewWriteableRequestResponseFrame(2, fakeData, fakeMetadata, 0),
	}

	var bytesWritten uint64
	for _, frame := range toBeWritten {
		n := frame.Len()
		err := tc.Write(frame)
		assert.NoError(t, err, "write failed")
		if frame.Header().Resumable() {
			bytesWritten += uint64(n)
		}
	}
	err := tc.Flush()
	assert.NoError(t, err)
	assert.Equal(t, bytesWritten, c.WriteBytes(), "write bytes doesn't match")
}

func TestTcpConn_Close(t *testing.T) {
	ctrl, nc, tc := InitMockTcpConn(t)
	defer ctrl.Finish()
	nc.EXPECT().Close().Return(fakeErr).Times(1)
	err := tc.Close()
	assert.Equal(t, fakeErr, err, "should return fake error")
}
