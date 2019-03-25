package transport

import (
	"bufio"
	"bytes"
	"context"
	"github.com/rsocket/rsocket-go/common"
	"github.com/rsocket/rsocket-go/common/logger"
	"github.com/rsocket/rsocket-go/framing"
	"io"
	"sync"
	"time"
)

var bytesOfKeepalive []byte

func init() {
	k := framing.NewFrameKeepalive(0, nil, true)
	defer k.Release()
	bf := &bytes.Buffer{}
	_, _ = common.NewUint24(k.Len()).WriteTo(bf)
	_, _ = k.WriteTo(bf)
	bytesOfKeepalive = bf.Bytes()
}

type tcpRConnection struct {
	c         io.ReadWriteCloser
	w         *bufio.Writer
	decoder   frameDecoder
	snd       chan framing.Frame
	onceClose *sync.Once
	done      chan struct{}
	handler   func(ctx context.Context, frame framing.Frame) error

	kaEnable      bool
	kaTicker      *time.Ticker
	kaInterval    time.Duration
	kaMaxLifetime time.Duration
	heartbeat     time.Time
}

func (p *tcpRConnection) Handle(handler func(ctx context.Context, frame framing.Frame) error) {
	p.handler = handler
}

func (p *tcpRConnection) Start(ctx context.Context) error {
	go func(ctx context.Context) {
		defer func() {
			_ = p.Close()
		}()
		p.loopSnd(ctx)
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
		p.kaTicker.Stop()
		err = p.c.Close()
	})
	return
}

func (p *tcpRConnection) loopSnd(ctx context.Context) {
	defer func() {
		logger.Debugf("send loop exit\n")
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
		case t := <-p.kaTicker.C:
			if t.Sub(p.heartbeat) > p.kaMaxLifetime {
				logger.Errorf("keepalive failed: remote connection is dead\n")
				stop = true
			}
			if !p.kaEnable {
				continue
			}
			if _, err := p.w.Write(bytesOfKeepalive); err != nil {
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
	p.heartbeat = time.Now()
	frameType := f.Header().Type()
	var frame framing.Frame
	switch frameType {
	case framing.FrameTypeSetup:
		frame = &framing.FrameSetup{BaseFrame: f}
	case framing.FrameTypeKeepalive:
		frame = &framing.FrameKeepalive{BaseFrame: f}
	case framing.FrameTypeRequestResponse:
		frame = &framing.FrameRequestResponse{BaseFrame: f}
	case framing.FrameTypeRequestFNF:
		frame = &framing.FrameFNF{BaseFrame: f}
	case framing.FrameTypeRequestStream:
		frame = &framing.FrameRequestStream{BaseFrame: f}
	case framing.FrameTypeRequestChannel:
		frame = &framing.FrameRequestChannel{BaseFrame: f}
	case framing.FrameTypeCancel:
		frame = &framing.FrameCancel{BaseFrame: f}
	case framing.FrameTypePayload:
		frame = &framing.FramePayload{BaseFrame: f}
	case framing.FrameTypeMetadataPush:
		frame = &framing.FrameMetadataPush{BaseFrame: f}
	case framing.FrameTypeError:
		frame = &framing.FrameError{BaseFrame: f}
	case framing.FrameTypeRequestN:
		frame = &framing.FrameRequestN{BaseFrame: f}
	case framing.FrameTypeLease:
		frame = &framing.FrameLease{BaseFrame: f}
	default:
		return common.ErrInvalidFrame
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
		if interval != p.kaInterval {
			p.kaTicker.Stop()
			p.kaTicker = time.NewTicker(interval)
		}
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

		kaEnable:      reportKeepalive,
		kaTicker:      time.NewTicker(keepaliveInterval),
		kaInterval:    keepaliveInterval,
		kaMaxLifetime: keepaliveMaxLifetime,
		heartbeat:     time.Now(),
	}
}
