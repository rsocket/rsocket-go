package transport

import (
	"bufio"
	"io"
	"net"
	"time"

	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/framing"
	"github.com/rsocket/rsocket-go/logger"
)

type tcpConn struct {
	rawConn net.Conn
	writer  *bufio.Writer
	decoder *LengthBasedFrameDecoder
	counter *Counter
}

func (p *tcpConn) SetCounter(c *Counter) {
	p.counter = c
}

func (p *tcpConn) SetDeadline(deadline time.Time) error {
	return p.rawConn.SetReadDeadline(deadline)
}

func (p *tcpConn) Read() (f framing.Frame, err error) {
	raw, err := p.decoder.Read()
	if err == io.EOF {
		return
	}
	if err != nil {
		err = errors.Wrap(err, "read frame failed")
		return
	}
	h := framing.ParseFrameHeader(raw)
	bf := common.NewByteBuff()
	_, err = bf.Write(raw[framing.HeaderLen:])
	if err != nil {
		err = errors.Wrap(err, "read frame failed")
		return
	}
	base := framing.NewBaseFrame(h, bf)
	if p.counter != nil && base.CanResume() {
		p.counter.incrReadBytes(base.Len())
	}
	f, err = framing.NewFromBase(base)
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

func (p *tcpConn) Flush() (err error) {
	err = p.writer.Flush()
	if err != nil {
		err = errors.Wrap(err, "flush failed")
	}
	return
}

func (p *tcpConn) Write(frame framing.Frame) (err error) {
	size := frame.Len()
	if p.counter != nil && frame.CanResume() {
		p.counter.incrWriteBytes(size)
	}
	_, err = common.MustNewUint24(size).WriteTo(p.writer)
	if err != nil {
		err = errors.Wrap(err, "write frame failed")
		return
	}
	var debugStr string
	if logger.IsDebugEnabled() {
		debugStr = frame.String()
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

func (p *tcpConn) Close() error {
	return p.rawConn.Close()
}

func newTCPRConnection(rawConn net.Conn) *tcpConn {
	return &tcpConn{
		rawConn: rawConn,
		writer:  bufio.NewWriter(rawConn),
		decoder: NewLengthBasedFrameDecoder(rawConn),
	}
}
