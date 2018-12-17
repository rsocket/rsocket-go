package rsocket

import (
	"bufio"
	"context"
	"io"
	"log"
	"sync"
)

type tcpRConnection struct {
	c        io.ReadWriteCloser
	decoder  FrameDecoder
	snd      chan Frame
	rcv      chan Frame
	handlers map[FrameType]interface{}
	once     *sync.Once
}

func (p *tcpRConnection) HandleFNF(h func(frame *FrameFNF) (err error)) {
	p.handlers[REQUEST_FNF] = h
}

func (p *tcpRConnection) HandlePayload(h func(*FramePayload) error) {
	p.handlers[PAYLOAD] = h
}

func (p *tcpRConnection) HandleRequestStream(h func(frame *FrameRequestStream) (err error)) {
	p.handlers[REQUEST_STREAM] = h
}

func (p *tcpRConnection) HandleRequestResponse(h func(frame *FrameRequestResponse) (err error)) {
	p.handlers[REQUEST_RESPONSE] = h
}

func (p *tcpRConnection) HandleSetup(h func(setup *FrameSetup) (err error)) {
	p.handlers[SETUP] = h
}

func (p *tcpRConnection) PostFlight(ctx context.Context) {
	go func() {
		if err := p.loopSnd(ctx); err != nil {
			log.Println("loop snd stopped:", err)
		}
	}()
	go func() {
		if err := p.loopRcv(ctx); err != nil {
			log.Println("loop rcv stopped:", err)
		}
	}()
}

func (p *tcpRConnection) Send(first Frame, others ...Frame) (err error) {
	defer func() {
		if e, ok := recover().(error); ok {
			err = e
		}
	}()
	p.snd <- first
	for _, other := range others {
		p.snd <- other
	}
	return
}

func (p *tcpRConnection) Close() (err error) {
	p.once.Do(func() {
		close(p.rcv)
		close(p.snd)
		err = p.c.Close()
	})
	return
}

func (p *tcpRConnection) loopRcv(ctx context.Context) error {
	defer func() {
		if err := p.Close(); err != nil {
			log.Println("close connection failed:", err)
		}
	}()
	return p.decoder.Handle(ctx, func(h *Header, raw []byte) error {
		switch h.Type() {
		case SETUP:
			f := &FrameSetup{}
			if err := f.Parse(h, raw); err != nil {
				return err
			}
			if h, ok := p.handlers[SETUP]; ok {
				if err := h.(func(*FrameSetup) error)(f); err != nil {
					return err
				}
			}
		case KEEPALIVE:
			f := &FrameKeepalive{}
			if err := f.Parse(h, raw); err != nil {
				return err
			}
			if h, ok := p.handlers[KEEPALIVE]; ok {
				if err := h.(func(*FrameKeepalive) error)(f); err != nil {
					return err
				}
			}
		case REQUEST_RESPONSE:
			f := &FrameRequestResponse{}
			if err := f.Parse(h, raw); err != nil {
				return err
			}
			if h, ok := p.handlers[REQUEST_RESPONSE]; ok {
				if err := h.(func(*FrameRequestResponse) error)(f); err != nil {
					return err
				}
			}
		case REQUEST_FNF:
			f := &FrameFNF{}
			if err := f.Parse(h, raw); err != nil {
				return err
			}
			if h, ok := p.handlers[REQUEST_FNF]; ok {
				if err := h.(func(*FrameFNF) error)(f); err != nil {
					return err
				}
			}
		case REQUEST_STREAM:
			f := &FrameRequestStream{}
			if err := f.Parse(h, raw); err != nil {
				return err
			}
			if h, ok := p.handlers[REQUEST_STREAM]; ok {
				if err := h.(func(*FrameRequestStream) error)(f); err != nil {
					return err
				}
			}
		case REQUEST_CHANNEL:
			f := &FrameRequestChannel{}
			if err := f.Parse(h, raw); err != nil {
				return err
			}
			if h, ok := p.handlers[REQUEST_CHANNEL]; ok {
				if err := h.(func(*FrameRequestChannel) error)(f); err != nil {
					return err
				}
			}
		case REQUEST_N:
			f := &FrameRequestN{}
			if err := f.Parse(h, raw); err != nil {
				return err
			}
			if h, ok := p.handlers[REQUEST_N]; ok {
				if err := h.(func(*FrameRequestN) error)(f); err != nil {
					return err
				}
			}
		case CANCEL:
			f := &FrameCancel{}
			if err := f.Parse(h, raw); err != nil {
				return err
			}
			if h, ok := p.handlers[CANCEL]; ok {
				if err := h.(func(*FrameCancel) error)(f); err != nil {
					return err
				}
			}
		case PAYLOAD:
			f := &FramePayload{}
			if err := f.Parse(h, raw); err != nil {
				return err
			}
			if h, ok := p.handlers[PAYLOAD]; ok {
				if err := h.(func(*FramePayload) error)(f); err != nil {
					return err
				}
			}
		case ERROR:
			f := &FrameError{}
			if err := f.Parse(h, raw); err != nil {
				return err
			}
			if h, ok := p.handlers[ERROR]; ok {
				if err := h.(func(*FrameError) error)(f); err != nil {
					return err
				}
			}
		case METADATA_PUSH:
			f := &FrameMetadataPush{}
			if err := f.Parse(h, raw); err != nil {
				return err
			}
			if h, ok := p.handlers[METADATA_PUSH]; ok {
				if err := h.(func(*FrameMetadataPush) error)(f); err != nil {
					return err
				}
			}
		case RESUME:
			panic("implement me")
		case RESUME_OK:
			panic("implement me")
		case EXT:
			panic("implement me")
		default:
			return ErrInvalidFrame
		}
		return nil
	})
}

func (p *tcpRConnection) loopSnd(ctx context.Context) error {
	if ctx == nil {
		return ErrInvalidContext
	}
	w := bufio.NewWriterSize(p.c, defaultBuffSize)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case frame, ok := <-p.snd:
			if !ok {
				return nil
			}
			if _, err := w.Write(encodeU24(frame.Size())); err != nil {
				return err
			}
			if _, err := frame.WriteTo(w); err != nil {
				return err
			}
			if err := w.Flush(); err != nil {
				return err
			}
		}
	}
}

func (p *tcpRConnection) handleKeepalive(f *FrameKeepalive) error {
	if !f.flags.Check(FlagRespond) {
		return nil
	}
	return p.Send(mkKeepalive(0, f.LastReceivedPosition(), f.Data()))
}

func newTcpRConnection(c io.ReadWriteCloser, buffSize int) (tc *tcpRConnection) {
	tc = &tcpRConnection{
		c:        c,
		decoder:  newLengthBasedFrameDecoder(c),
		snd:      make(chan Frame, buffSize),
		rcv:      make(chan Frame, buffSize),
		once:     &sync.Once{},
		handlers: make(map[FrameType]interface{}),
	}
	tc.handlers[KEEPALIVE] = tc.handleKeepalive
	return
}
