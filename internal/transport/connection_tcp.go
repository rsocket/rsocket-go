package transport

import (
	"bufio"
	"net"
	"time"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/framing"
	"github.com/rsocket/rsocket-go/internal/logger"
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
	if err != nil {
		return
	}
	h := framing.ParseFrameHeader(raw)
	bf := common.BorrowByteBuffer()
	_, err = bf.Write(raw[framing.HeaderLen:])
	if err != nil {
		common.ReturnByteBuffer(bf)
		return
	}
	base := framing.NewBaseFrame(h, bf)
	if p.counter != nil && base.IsResumable() {
		p.counter.incrReadBytes(base.Len())
	}
	f, err = framing.NewFromBase(base)
	if err != nil {
		common.ReturnByteBuffer(bf)
		return
	}
	err = f.Validate()
	if err != nil {
		common.ReturnByteBuffer(bf)
		return
	}
	if logger.IsDebugEnabled() {
		logger.Debugf("<--- rcv: %s\n", f)
	}
	return
}

func (p *tcpConn) Write(frame framing.Frame) error {
	size := frame.Len()
	if p.counter != nil && frame.IsResumable() {
		p.counter.incrWriteBytes(size)
	}
	if _, err := common.NewUint24(size).WriteTo(p.writer); err != nil {
		return err
	}
	if _, err := frame.WriteTo(p.writer); err != nil {
		return err
	}
	if logger.IsDebugEnabled() {
		logger.Debugf("---> snd: %s\n", frame)
	}
	return p.writer.Flush()
}

func (p *tcpConn) Close() error {
	return p.rawConn.Close()
}

func newTCPRConnection(rawConn net.Conn) *tcpConn {
	return &tcpConn{
		rawConn: rawConn,
		writer:  bufio.NewWriterSize(rawConn, tcpWriteBuffSize),
		decoder: NewLengthBasedFrameDecoder(rawConn),
	}
}
