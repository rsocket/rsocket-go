package transport

import (
	"bufio"
	"context"
	"io"
	"sync"
	"time"

	"github.com/rsocket/rsocket-go/common"
	"github.com/rsocket/rsocket-go/common/logger"
	"github.com/rsocket/rsocket-go/framing"
)

type tcpRConnection struct {
	c         io.ReadWriteCloser
	w         *bufio.Writer
	decoder   frameDecoder
	snd       chan framing.Frame
	onceClose *sync.Once
	done      chan struct{}
	handler   func(ctx context.Context, frame framing.Frame) error

	kaEnable bool
	ka       *keepaliver
}

func (p *tcpRConnection) Handle(handler func(ctx context.Context, frame framing.Frame) error) {
	p.handler = handler
}

func (p *tcpRConnection) Start(ctx context.Context) error {
	go func(ctx context.Context) {
		defer func() {
			_ = p.Close()
		}()
		p.loopWrite(ctx)
	}(ctx)
	defer func() {
		_ = p.Close()
	}()
	return p.decoder.handle(func(raw []byte) (err error) {
		defer func() {
			if e := recover(); e != nil {
				logger.Errorf("decoder exit: %s\n", e)
			}
		}()
		h := framing.ParseFrameHeader(raw)
		bf := common.BorrowByteBuffer()
		_, err = bf.Write(raw[framing.HeaderLen:])
		if err == nil {
			err = p.onRcv(ctx, framing.NewBaseFrame(h, bf))
		}
		return
	})
}
func (p *tcpRConnection) Write(frame framing.Frame) error {
	defer frame.Release()
	if _, err := common.NewUint24(frame.Len()).WriteTo(p.w); err != nil {
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

func (p *tcpRConnection) Send(frame framing.Frame) (err error) {
	defer func() {
		if e, ok := recover().(error); ok {
			err = e
		}
	}()
	p.snd <- frame
	return
}

func (p *tcpRConnection) Close() (err error) {
	p.onceClose.Do(func() {
		close(p.snd)
		<-p.done
		_ = p.ka.Close()
		err = p.c.Close()
	})
	return
}

func (p *tcpRConnection) loopWrite(ctx context.Context) {
	defer func() {
		close(p.done)
	}()

	var stop bool
	for {
		if stop {
			break
		}
		select {
		case <-ctx.Done():
			if err := ctx.Err(); err != nil {
				logger.Errorf("send loop exit: %s\n", err)
			}
			return
		case t := <-p.ka.C():
			if p.ka.IsDead(t) {
				logger.Errorf("keepalive failed: remote connection is dead\n")
				stop = true
				continue
			}
			if !p.kaEnable {
				continue
			}
			if _, err := p.w.Write(getKeepaliveBytes(true)); err != nil {
				logger.Errorf("send frame failed: %s\n", err)
			} else if err := p.w.Flush(); err != nil {
				logger.Errorf("send frame failed: %s\n", err)
			}
		case out, ok := <-p.snd:
			if !ok {
				stop = true
				break
			}
			if err := p.Write(out); err != nil {
				logger.Errorf("send frame failed: %s\n", err)
			}
		}
	}
}

func (p *tcpRConnection) onRcv(ctx context.Context, f *framing.BaseFrame) (err error) {
	p.ka.Heartbeat()
	var frame framing.Frame
	frame, err = unwindFrame(f)
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

	if setupFrame, ok := frame.(*framing.FrameSetup); ok {
		interval := setupFrame.TimeBetweenKeepalive()
		p.ka.Reset(interval)
	}
	err = p.handler(ctx, frame)
	if err != nil {
		logger.Warnf("handle frame %s failed: %s\n", frame.Header().Type(), err.Error())
	}
	return
}

func newTCPRConnection(c io.ReadWriteCloser, keepaliveInterval, keepaliveMaxLifetime time.Duration, reportKeepalive bool) *tcpRConnection {
	return &tcpRConnection{
		c:         c,
		w:         bufio.NewWriterSize(c, common.DefaultTCPWriteBuffSize),
		decoder:   newLengthBasedFrameDecoder(c),
		snd:       make(chan framing.Frame, common.DefaultTCPSendChanSize),
		done:      make(chan struct{}),
		onceClose: &sync.Once{},
		kaEnable:  reportKeepalive,
		ka:        newKeepaliver(keepaliveInterval, keepaliveMaxLifetime),
	}
}
