package rsocket

import (
	"bufio"
	"context"
	"io"
	"sync"
	"time"
)

const (
	writeBuffSize = 16 * 1024
	sndChanSize   = 64
)

type tcpRConnection struct {
	c         io.ReadWriteCloser
	w         *bufio.Writer
	decoder   frameDecoder
	snd       chan Frame
	onceClose *sync.Once
	done      chan struct{}
	handler   func(ctx context.Context, frame Frame) error
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
		err = p.c.Close()
	})
	return
}

func (p *tcpRConnection) loopSnd(ctx context.Context) {
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
				logger.Errorf("send loop end: %s\n", err)
			}
			return
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
	now := time.Now()
	defer func() {
		logger.Debugf("handle %s: cost=%s\n", f.header.Type(), time.Now().Sub(now))
	}()
	return p.handler(ctx, frame)
}

func newTcpRConnection(c io.ReadWriteCloser) *tcpRConnection {
	return &tcpRConnection{
		c:         c,
		w:         bufio.NewWriterSize(c, writeBuffSize),
		decoder:   newLengthBasedFrameDecoder(c),
		snd:       make(chan Frame, sndChanSize),
		done:      make(chan struct{}),
		onceClose: &sync.Once{},
	}
}
