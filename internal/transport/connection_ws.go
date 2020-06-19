package transport

import (
	"bytes"
	"io"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/framing"
	"github.com/rsocket/rsocket-go/logger"
)

var _buffPool = sync.Pool{
	New: func() interface{} { return &bytes.Buffer{} },
}

type wsConn struct {
	c       *websocket.Conn
	counter *Counter
}

func (p *wsConn) SetCounter(c *Counter) {
	p.counter = c
}

func (p *wsConn) SetDeadline(deadline time.Time) error {
	return p.c.SetReadDeadline(deadline)
}

func (p *wsConn) Read() (f framing.Frame, err error) {
	t, raw, err := p.c.ReadMessage()
	if err != nil {
		err = errors.Wrap(err, "read frame failed")
		return
	}
	if t != websocket.BinaryMessage {
		logger.Warnf("omit non-binary message %d\n", t)
		return p.Read()
	}
	// validate min length
	if len(raw) < framing.HeaderLen {
		err = errors.Wrap(ErrIncompleteHeader, "read frame failed")
		return
	}
	header := framing.ParseFrameHeader(raw)
	bf := common.NewByteBuff()
	_, err = bf.Write(raw[framing.HeaderLen:])
	if err != nil {
		err = errors.Wrap(err, "read frame failed")
		return
	}
	base := framing.NewRawFrame(header, bf)
	f, err = framing.FromRawFrame(base)
	if err != nil {
		err = errors.Wrap(err, "read frame failed")
		return
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

func (p *wsConn) Flush() (err error) {
	return
}

func (p *wsConn) Write(frame framing.FrameSupport) (err error) {
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
	if logger.IsDebugEnabled() {
		logger.Debugf("---> snd: %s\n", frame)
	}
	return
}

func (p *wsConn) Close() error {
	return p.c.Close()
}

func newWebsocketConnection(rawConn *websocket.Conn) *wsConn {
	return &wsConn{
		c: rawConn,
	}
}
