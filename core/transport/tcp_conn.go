package transport

import (
	"bufio"
	"io"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/logger"
)

type TcpConn struct {
	conn    net.Conn
	writer  *bufio.Writer
	decoder *LengthBasedFrameDecoder
	counter *core.TrafficCounter
}

func (p *TcpConn) SetCounter(c *core.TrafficCounter) {
	p.counter = c
}

func (p *TcpConn) SetDeadline(deadline time.Time) error {
	return p.conn.SetReadDeadline(deadline)
}

func (p *TcpConn) Read() (f core.Frame, err error) {
	raw, err := p.decoder.Read()
	if err == io.EOF {
		return
	}
	if err != nil {
		err = errors.Wrap(err, "read frame failed")
		return
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

func (p *TcpConn) Flush() (err error) {
	err = p.writer.Flush()
	if err != nil {
		err = errors.Wrap(err, "flush failed")
	}
	return
}

func (p *TcpConn) Write(frame core.WriteableFrame) (err error) {
	size := frame.Len()
	if p.counter != nil && frame.Header().Resumable() {
		p.counter.IncWriteBytes(size)
	}
	_, err = common.MustNewUint24(size).WriteTo(p.writer)
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
		logger.Debugf("---> snd: %s\n", debugStr)
	}
	return
}

func (p *TcpConn) Close() error {
	return p.conn.Close()
}

func NewTcpConn(conn net.Conn) *TcpConn {
	return &TcpConn{
		conn:    conn,
		writer:  bufio.NewWriter(conn),
		decoder: NewLengthBasedFrameDecoder(conn),
	}
}
