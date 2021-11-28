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

// RawWebsocketConn is Raw websocket connection.
// Only for mock tests.
type RawWebsocketConn interface {
	io.Closer
	// SetReadDeadline set read deadline.
	SetReadDeadline(time.Time) error
	// ReadMessage reads next message.
	ReadMessage() (messageType int, p []byte, err error)
	// WriteMessage writes next message.
	WriteMessage(messageType int, data []byte) error
}

// WebsocketConn is websocket RSocket connection.
type WebsocketConn struct {
	c       RawWebsocketConn
	counter *core.TrafficCounter
}

// Addr returns the address info.
func (p *WebsocketConn) Addr() string {
	if c, ok := p.c.(*websocket.Conn); ok {
		return c.RemoteAddr().String()
	}
	return ""
}

// SetCounter bind a counter which can count r/w bytes.
func (p *WebsocketConn) SetCounter(c *core.TrafficCounter) {
	p.counter = c
}

// SetDeadline set deadline for current connection.
// After this deadline, connection will be closed.
func (p *WebsocketConn) SetDeadline(deadline time.Time) error {
	return p.c.SetReadDeadline(deadline)
}

// Read reads next frame from Conn.
func (p *WebsocketConn) Read() (f core.BufferedFrame, err error) {
	t, raw, err := p.c.ReadMessage()

	if err == io.EOF {
		return
	}

	if websocket.IsCloseError(err) || websocket.IsUnexpectedCloseError(err) || isClosedErr(err) {
		err = io.EOF
		return
	}

	if err != nil {
		err = errors.Wrap(err, "read frame failed")
		return
	}

	// Skip non-binary message
	if t != websocket.BinaryMessage {
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
		logger.Debugf("%s\n", framing.PrintFrame(f))
	}
	return
}

// Flush flush data.
func (p *WebsocketConn) Flush() (err error) {
	return
}

// Write writes frames.
func (p *WebsocketConn) Write(frame core.WriteableFrame) (err error) {
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
		logger.Debugf("%s\n", framing.PrintFrame(frame))
	}
	return
}

// Close closes connection.
func (p *WebsocketConn) Close() error {
	return p.c.Close()
}

// NewWebsocketConnection creates a new RSocket websocket connection.
func NewWebsocketConnection(rawConn RawWebsocketConn) *WebsocketConn {
	return &WebsocketConn{
		c: rawConn,
	}
}
