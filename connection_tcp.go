package rsocket

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"sync"
	"time"
)

var errIncompleteFrame = errors.New("incomplete frame")

var bytesOfKeepalive []byte

func init() {
	k := createKeepalive(0, nil, true)
	defer k.Release()
	bf := &bytes.Buffer{}
	_, _ = newUint24(k.Len()).WriteTo(bf)
	_, _ = k.WriteTo(bf)
	bytesOfKeepalive = bf.Bytes()
}

type tcpRConnection struct {
	c         io.ReadWriteCloser
	w         *bufio.Writer
	decoder   frameDecoder
	snd       chan Frame
	onceClose *sync.Once
	done      chan struct{}
	handler   func(ctx context.Context, frame Frame) error

	kaEnable      bool
	kaTicker      *time.Ticker
	kaInterval    time.Duration
	kaMaxLifetime time.Duration
	heartbeat     time.Time
}

func (p *tcpRConnection) Handle(handler func(ctx context.Context, frame Frame) error) {
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
	return p.decoder.handle(func(raw []byte) error {
		defer func() {
			_ = recover()
		}()
		h := parseHeaderBytes(raw)
		bf := borrowByteBuffer()
		if _, err := bf.Write(raw[headerLen:]); err != nil {
			return err
		}
		return p.onRcv(ctx, &baseFrame{h, bf})
	})
}
func (p *tcpRConnection) Write(frame Frame) error {
	defer frame.Release()
	if _, err := newUint24(frame.Len()).WriteTo(p.w); err != nil {
		return err
	}
	if _, err := frame.WriteTo(p.w); err != nil {
		return err
	}
	return p.w.Flush()
}

func (p *tcpRConnection) Send(frame Frame) (err error) {
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
		// TODO: release unfinished frame
		<-p.done
		p.kaTicker.Stop()
		err = p.c.Close()
	})
	return
}

func (p *tcpRConnection) loopSnd(ctx context.Context) {
	defer func() {
		logger.Debugf("connection send loop end\n")
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
				logger.Errorf("send loop end: %s\n", err)
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

func (p *tcpRConnection) onRcv(ctx context.Context, f *baseFrame) error {
	p.heartbeat = time.Now()
	var frame Frame
	switch f.header.Type() {
	case tSetup:
		frame = &frameSetup{f}
	case tKeepalive:
		frame = &frameKeepalive{f}
	case tRequestResponse:
		frame = &frameRequestResponse{f}
	case tRequestFNF:
		frame = &frameFNF{f}
	case tRequestStream:
		frame = &frameRequestStream{f}
	case tRequestChannel:
		frame = &frameRequestChannel{f}
	case tCancel:
		frame = &frameCancel{f}
	case tPayload:
		frame = &framePayload{f}
	case tMetadataPush:
		frame = &frameMetadataPush{f}
	case tError:
		frame = &frameError{f}
	case tRequestN:
		frame = &frameRequestN{f}
	default:
		return ErrInvalidFrame
	}

	if err := frame.validate(); err != nil {
		return err
	}

	if setupFrame, ok := frame.(*frameSetup); ok {
		interval := setupFrame.TimeBetweenKeepalive()
		if interval != p.kaInterval {
			p.kaTicker.Stop()
			p.kaTicker = time.NewTicker(interval)
		}
	}

	//now := time.Now()
	//defer func() {
	//	logger.Debugf("handle %s: cost=%s\n", f.header.Type(), time.Now().Sub(now))
	//}()
	return p.handler(ctx, frame)
}

func newTCPRConnection(c io.ReadWriteCloser, keepaliveInterval, keepaliveMaxLifetime time.Duration, reportKeepalive bool) *tcpRConnection {
	return &tcpRConnection{
		c:         c,
		w:         bufio.NewWriterSize(c, defaultConnTcpWriteBuffSize),
		decoder:   newLengthBasedFrameDecoder(c),
		snd:       make(chan Frame, defaultConnSendChanSize),
		done:      make(chan struct{}),
		onceClose: &sync.Once{},

		kaEnable:      reportKeepalive,
		kaTicker:      time.NewTicker(keepaliveInterval),
		kaInterval:    keepaliveInterval,
		kaMaxLifetime: keepaliveMaxLifetime,
		heartbeat:     time.Now(),
	}
}
