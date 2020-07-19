package transport_test

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/logger"
	"github.com/stretchr/testify/assert"
)

func InitMockWsConn(t *testing.T) (*gomock.Controller, *mockRawWsConn, *transport.WsConn) {
	ctrl := gomock.NewController(t)
	rawConn := newMockRawWsConn(ctrl)
	conn := transport.NewWebsocketConnection(rawConn)
	return ctrl, rawConn, conn
}

func TestWsConn_Read_Empty(t *testing.T) {
	ctrl, rawConn, conn := InitMockWsConn(t)
	defer ctrl.Finish()

	rawConn.EXPECT().ReadMessage().Return(0, nil, fakeErr).AnyTimes()
	_, err := conn.Read()
	assert.Error(t, err, "should read failed")
}

func TestWsConn_Read(t *testing.T) {
	ctrl, rc, wc := InitMockWsConn(t)
	defer ctrl.Finish()

	c := core.NewCounter()
	wc.SetCounter(c)

	toBeWritten := []core.WriteableFrame{
		framing.NewWriteablePayloadFrame(1, fakeData, fakeMetadata, 0),
		framing.NewWriteableKeepaliveFrame(0, fakeData, true),
		framing.NewWriteableRequestResponseFrame(2, fakeData, fakeMetadata, 0),
	}

	var (
		writtenBytesSlice [][]byte
		writtenBytes      int
	)

	for _, frame := range toBeWritten {
		n := frame.Len()
		if frame.Header().Resumable() {
			writtenBytes += n
		}
		b := &bytes.Buffer{}
		_, err := frame.WriteTo(b)
		assert.NoError(t, err, "write frame failed")
		writtenBytesSlice = append(writtenBytesSlice, b.Bytes())
	}

	cursor := 0
	rc.EXPECT().
		ReadMessage().
		DoAndReturn(func() (int, []byte, error) {
			defer func() {
				cursor++
			}()
			if cursor >= len(writtenBytesSlice) {
				return 0, nil, io.EOF
			}
			return websocket.BinaryMessage, writtenBytesSlice[cursor], nil
		}).
		AnyTimes()

	var results []core.Frame
	for {
		next, err := wc.Read()
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

func TestWsConn_SetDeadline(t *testing.T) {
	ctrl, mc, c := InitMockWsConn(t)
	defer ctrl.Finish()

	mc.EXPECT().SetReadDeadline(gomock.Any()).Times(1)
	err := c.SetDeadline(time.Now())
	assert.NoError(t, err, "call setDeadline failed")
}

func TestWsConn_Flush_Nothing(t *testing.T) {
	ctrl, mc, wc := InitMockWsConn(t)
	defer ctrl.Finish()

	c := core.NewCounter()
	wc.SetCounter(c)

	mc.EXPECT().WriteMessage(websocket.BinaryMessage, gomock.Any()).Times(0)

	err := wc.Flush()
	assert.NoError(t, err, "flush failed")
	assert.Equal(t, 0, int(c.WriteBytes()), "bytes written should be zero")
}

func TestWsConn_WriteWithBrokenConn(t *testing.T) {
	logger.SetLogger(nil)
	logger.SetLevel(logger.LevelDebug)
	ctrl, mc, wc := InitMockWsConn(t)
	defer ctrl.Finish()
	mc.EXPECT().
		WriteMessage(websocket.BinaryMessage, gomock.Any()).
		Return(fakeErr).
		AnyTimes()
	err := wc.Write(framing.NewWriteablePayloadFrame(1, fakeData, fakeMetadata, 0))
	assert.Equal(t, fakeErr, errors.Cause(err), "should be fake error")
}

func TestWsConn_Write(t *testing.T) {
	ctrl, mc, wc := InitMockWsConn(t)
	defer ctrl.Finish()

	c := core.NewCounter()
	wc.SetCounter(c)

	toBeWritten := []core.WriteableFrame{
		framing.NewWriteablePayloadFrame(1, fakeData, fakeMetadata, 0),
		framing.NewWriteableKeepaliveFrame(0, fakeData, true),
		framing.NewWriteableRequestResponseFrame(2, fakeData, fakeMetadata, 0),
	}

	mc.EXPECT().
		WriteMessage(websocket.BinaryMessage, gomock.Any()).
		Return(nil).
		Times(len(toBeWritten))

	var bytesWritten uint64
	for _, frame := range toBeWritten {
		n := frame.Len()
		err := wc.Write(frame)
		assert.NoError(t, err, "write failed")
		if frame.Header().Resumable() {
			bytesWritten += uint64(n)
		}
	}
	err := wc.Flush()
	assert.NoError(t, err)
	assert.Equal(t, bytesWritten, c.WriteBytes(), "write bytes doesn't match")
}

func TestWsConn_Close(t *testing.T) {
	ctrl, mc, wc := InitMockWsConn(t)
	defer ctrl.Finish()
	mc.EXPECT().Close().Return(fakeErr).Times(1)
	err := wc.Close()
	assert.Equal(t, fakeErr, err, "should return fake error")
}
