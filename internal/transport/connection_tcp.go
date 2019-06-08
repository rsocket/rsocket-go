package transport

import (
	"bufio"
	"context"
	"net"
	"sync"
	"time"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/framing"
	"github.com/rsocket/rsocket-go/internal/logger"
)

type tcpRConnection struct {
	c       net.Conn
	w       *bufio.Writer
	r       FrameDecoder
	handler func(ctx context.Context, frame framing.Frame) error
	once    *sync.Once
	counter *Counter
}

func (p *tcpRConnection) SetCounter(c *Counter) {
	p.counter = c
}

func (p *tcpRConnection) SetDeadline(deadline time.Time) error {
	return p.c.SetDeadline(deadline)
}

func (p *tcpRConnection) Handle(handler func(ctx context.Context, frame framing.Frame) error) {
	p.handler = handler
}

func (p *tcpRConnection) Start(ctx context.Context) (err error) {
	defer func() {
		_ = p.Close()
	}()
	err = p.r.Handle(func(raw []byte) (err error) {
		h := framing.ParseFrameHeader(raw)
		bf := common.BorrowByteBuffer()
		_, err = bf.Write(raw[framing.HeaderLen:])
		if err == nil {
			err = p.onRcv(ctx, framing.NewBaseFrame(h, bf))
		}
		return
	})
	return
}

func (p *tcpRConnection) Write(frame framing.Frame) error {
	size := frame.Len()
	if p.counter != nil && frame.IsResumable() {
		p.counter.incrWriteBytes(size)
	}
	if _, err := common.NewUint24(size).WriteTo(p.w); err != nil {
		return err
	}
	if _, err := frame.WriteTo(p.w); err != nil {
		return err
	}
	if logger.IsDebugEnabled() {
		logger.Debugf("---> snd: %s\n", frame)
	}
	return p.w.Flush()
}

func (p *tcpRConnection) Close() (err error) {
	p.once.Do(func() {
		err = p.c.Close()
	})
	return
}

func (p *tcpRConnection) onRcv(ctx context.Context, f *framing.BaseFrame) (err error) {
	if p.counter != nil && f.IsResumable() {
		p.counter.incrReadBytes(f.Len())
	}
	var frame framing.Frame
	frame, err = framing.NewFromBase(f)
	if err != nil {
		return
	}
	err = frame.Validate()
	if err != nil {
		return
	}
	if logger.IsDebugEnabled() {
		logger.Debugf("<--- rcv: %s\n", frame)
	}
	err = p.handler(ctx, frame)
	if err != nil {
		logger.Warnf("handle frame %s failed: %s\n", frame.Header().Type(), err.Error())
	}
	return
}

func newTCPRConnection(rawConn net.Conn) *tcpRConnection {
	return &tcpRConnection{
		c:    rawConn,
		w:    bufio.NewWriterSize(rawConn, common.DefaultTCPWriteBuffSize),
		r:    NewLengthBasedFrameDecoder(rawConn),
		once: &sync.Once{},
	}
}
