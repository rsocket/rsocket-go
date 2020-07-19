package transport

import (
	"bytes"
	"io"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/logger"
)

var _buffPool = sync.Pool{
	New: func() interface{} { return &bytes.Buffer{} },
}

type RawWsConn interface {
	io.Closer
	SetReadDeadline(time.Time) error
	ReadMessage() (messageType int, p []byte, err error)
	WriteMessage(messageType int, data []byte) error
}

type WsConn struct {
	c       RawWsConn
	counter *core.Counter
}

func (p *WsConn) SetCounter(c *core.Counter) {
	p.counter = c
}

func (p *WsConn) SetDeadline(deadline time.Time) error {
	return p.c.SetReadDeadline(deadline)
}

func (p *WsConn) Read() (f core.Frame, err error) {
	t, raw, err := p.c.ReadMessage()
	if err == io.EOF {
		return
	}
	if err != nil {
		err = errors.Wrap(err, "read frame failed")
		return
	}
	if t != websocket.BinaryMessage {
		logger.Warnf("omit non-binary message %d\n", t)
		return p.Read()
	}

	f, err = framing.FromBytes(raw)
	if err != nil {
		err = errors.Wrap(err, "read frame failed")
		return
	}

	if p.counter != nil && f.Header().Resumable() {
		p.counter.IncReadBytes(f.Len())
	}

	err = f.Validate()
	if err != nil {
		err = errors.Wrap(err, "read frame failed")
		return
	}
	if logger.IsDebugEnabled() {
		logger.Debugf("<--- rcv: %s\n", f)
	}
	return
}

func (p *WsConn) Flush() (err error) {
	return
}

func (p *WsConn) Write(frame core.WriteableFrame) (err error) {
	size := frame.Len()
	bf := _buffPool.Get().(*bytes.Buffer)
	defer func() {
		bf.Reset()
		_buffPool.Put(bf)
	}()
	_, err = frame.WriteTo(bf)
	if err != nil {
		return
	}
	err = p.c.WriteMessage(websocket.BinaryMessage, bf.Bytes())
	if err == io.EOF {
		return
	}
	if err != nil {
		err = errors.Wrap(err, "write frame failed")
		return
	}
	if p.counter != nil && frame.Header().Resumable() {
		p.counter.IncWriteBytes(size)
	}
	if logger.IsDebugEnabled() {
		logger.Debugf("---> snd: %s\n", frame)
	}
	return
}

func (p *WsConn) Close() error {
	return p.c.Close()
}

func NewWebsocketConnection(rawConn RawWsConn) *WsConn {
	return &WsConn{
		c: rawConn,
	}
}
