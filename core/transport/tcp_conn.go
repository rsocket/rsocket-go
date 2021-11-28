package transport

import (
	"bufio"
	"io"
	"net"
	"time"

	"github.com/pkg/errors"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/internal/u24"
	"github.com/rsocket/rsocket-go/logger"
)

// TCPConn is RSocket connection for TCP transport.
type TCPConn struct {
	conn    net.Conn
	writer  *bufio.Writer
	decoder *LengthBasedFrameDecoder
	counter *core.TrafficCounter
}

func (p TCPConn) Addr() string {
	addr := p.conn.RemoteAddr()
	return addr.String()
}

// SetCounter bind a counter which can count r/w bytes.
func (p *TCPConn) SetCounter(c *core.TrafficCounter) {
	p.counter = c
}

// SetDeadline set deadline for current connection.
// After this deadline, connection will be closed.
func (p *TCPConn) SetDeadline(deadline time.Time) error {
	return p.conn.SetReadDeadline(deadline)
}

// Read reads next frame from Conn.
func (p *TCPConn) Read() (f core.BufferedFrame, err error) {
	raw, err := p.decoder.Read()
	if err == io.EOF {
		return
	}
	if err != nil {
		err = errors.Wrap(err, "read frame failed:")
		return
	}
	f, err = framing.FromBytes(raw)
	if err != nil {
		err = errors.Wrap(err, "decode frame failed:")
		return
	}
	if p.counter != nil && f.Header().Resumable() {
		p.counter.IncReadBytes(f.Len())
	}
	err = f.Validate()
	if err != nil {
		err = errors.Wrap(err, "validate frame failed:")
		return
	}
	if logger.IsDebugEnabled() {
		logger.Debugf("%s\n", framing.PrintFrame(f))
	}
	return
}

// Flush flush data.
func (p *TCPConn) Flush() (err error) {
	err = p.writer.Flush()
	if err != nil {
		err = errors.Wrap(err, "flush failed")
	}
	return
}

// Write writes a frame.
func (p *TCPConn) Write(frame core.WriteableFrame) (err error) {
	size := frame.Len()
	if p.counter != nil && frame.Header().Resumable() {
		p.counter.IncWriteBytes(size)
	}
	_, err = u24.MustNewUint24(size).WriteTo(p.writer)
	if err != nil {
		err = errors.Wrap(err, "write frame failed")
		return
	}
	var debugStr string
	if logger.IsDebugEnabled() {
		debugStr = framing.PrintFrame(frame)
	}
	_, err = frame.WriteTo(p.writer)
	if err != nil {
		err = errors.Wrap(err, "write frame failed")
		return
	}
	if logger.IsDebugEnabled() {
		logger.Debugf("%s\n", debugStr)
	}
	return
}

// Close closes current connection.
func (p *TCPConn) Close() error {
	return p.conn.Close()
}

// NewTCPConn creates a new TCP RSocket connection.
func NewTCPConn(conn net.Conn) *TCPConn {
	return &TCPConn{
		conn:    conn,
		writer:  bufio.NewWriterSize(conn, 8192),
		decoder: NewLengthBasedFrameDecoder(conn),
	}
}
