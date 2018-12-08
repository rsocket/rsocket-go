package rsocket

import (
	"bufio"
	"context"
	"github.com/pkg/errors"
	"io"
	"log"
	"sync"
)

var errUnsupportedFrame = errors.New("unsupported frame")

type tcpRConnection struct {
	c          io.ReadWriteCloser
	decoder    FrameDecoder
	snd        chan Frame
	rcv        chan Frame
	hSetup     func(*FrameSetup) error
	hReqRes    func(*FrameRequestResponse) error
	hReqStream func(*FrameRequestStream) error
	hPayload   func(*FramePayload) error
	hFNF       func(*FrameFNF) error
	once       *sync.Once
}

func (p *tcpRConnection) HandleFNF(callback func(frame *FrameFNF) (err error)) {
	p.hFNF = callback
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

func (p *tcpRConnection) HandleRequestStream(callback func(frame *FrameRequestStream) (err error)) {
	p.hReqStream = callback
}

func (p *tcpRConnection) HandleRequestResponse(callback func(frame *FrameRequestResponse) (err error)) {
	p.hReqRes = callback
}

func (p *tcpRConnection) HandleSetup(h func(setup *FrameSetup) (err error)) {
	p.hSetup = h
}

func (p *tcpRConnection) Close() (err error) {
	p.once.Do(func() {
		close(p.snd)
		err = p.c.Close()
	})
	return
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

func (p *tcpRConnection) HandleCancel(f *FrameCancel) error {
	return nil
}

func (p *tcpRConnection) HandleError(f *FrameError) error {
	return nil
}

func (p *tcpRConnection) HandleLease(f *FrameLease) error {
	return nil
}

func (p *tcpRConnection) HandleRequestChannel(f *FrameRequestChannel) error {
	return nil
}

func (p *tcpRConnection) HandleRequestN(f *FrameRequestN) error {
	return nil
}

func (p *tcpRConnection) HandlePayload(h func(*FramePayload) error) {
	p.hPayload = h
}

func (p *tcpRConnection) HandleMetadataPush(f *FrameMetadataPush) error {
	return nil
}

func (p *tcpRConnection) loopRcv(c context.Context) error {
	ctx, cancel := context.WithCancel(c)
	defer func() {
		cancel()
		if err := p.Close(); err != nil {
			log.Println("close connection failed:", err)
		}
	}()
	return p.decoder.Handle(ctx, func(h *Header, raw []byte) (err error) {
		t := h.Type()
		switch t {
		case SETUP:
			if p.hSetup != nil {
				err = p.hSetup(asSetup(h, raw))
			}
		case LEASE:
			err = p.HandleLease(asLease(h, raw))
		case KEEPALIVE:
			err = p.handleKeepalive(asKeepalive(h, raw))
		case REQUEST_RESPONSE:
			err = p.hReqRes(asRequestResponse(h, raw))
		case REQUEST_FNF:
			if p.hFNF != nil {
				err = p.hFNF(asFNF(h, raw))
			}
		case REQUEST_STREAM:
			err = p.hReqStream(asRequestStream(h, raw))
		case REQUEST_CHANNEL:
			err = p.HandleRequestChannel(asRequestChannel(h, raw))
		case REQUEST_N:
			err = p.HandleRequestN(asRequestN(h, raw))
		case CANCEL:
			err = p.HandleCancel(asCancel(h, raw))
		case PAYLOAD:
			if p.hPayload != nil {
				err = p.hPayload(asPayload(h, raw))
			}
		case ERROR:
			err = p.HandleError(asError(h, raw))
		case METADATA_PUSH:
			err = p.HandleMetadataPush(asMetadataPush(h, raw))
		//case RESUME:
		//case RESUME_OK:
		//case EXT:
		//	err = p.HandleExtension(&FrameExtension{frame})
		default:
			return errUnsupportedFrame
		}
		return
	})
}

func (p *tcpRConnection) loopSnd(ctx context.Context) error {
	w := bufio.NewWriterSize(p.c, defaultBuffSize)
	for frame := range p.snd {
		bs := frame.Bytes()
		frameLength := encodeU24(len(bs))
		if _, err := w.Write(frameLength); err != nil {
			return err
		}
		if _, err := w.Write(bs); err != nil {
			return err
		}
		if err := w.Flush(); err != nil {
			return err
		}
	}
	return nil
}

func (p *tcpRConnection) handleKeepalive(f *FrameKeepalive) error {
	if !f.flags.Check(FlagRespond) {
		return nil
	}
	return p.Send(mkKeepalive(0, f.LastReceivedPosition(), f.Data()))
}

func newTcpRConnection(c io.ReadWriteCloser, buffSize int) *tcpRConnection {
	return &tcpRConnection{
		c: c,

		decoder: newLengthBasedFrameDecoder(c),
		snd:     make(chan Frame, buffSize),
		rcv:     make(chan Frame, buffSize),
		once:    &sync.Once{},
	}
}
