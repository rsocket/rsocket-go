package protocol

import (
	"context"
	"github.com/pkg/errors"
	"log"
	"net"
)

var errUnknownFrame = errors.New("bad frame")

type tcpRConnection struct {
	c       net.Conn
	decoder FrameDecoder
	snd     chan Frame

	handlerSetup func(setup *FrameSetup) error
}

func (p *tcpRConnection) HandleSetup(h func(setup *FrameSetup) (err error)) {
	p.handlerSetup = h
}

func (p *tcpRConnection) Close() error {
	close(p.snd)
	return p.c.Close()
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

func (p *tcpRConnection) HandleRequestResponse(f *FrameRequestResponse) error {
	base := &BaseFrame{
		StreamID: f.StreamID(),
		Flags:    FlagMetadata | FlagComplete | FlagNext,
		Type:     PAYLOAD,
	}
	return p.Send(NewPayload(base, []byte("pong"), f.Metadata()))
}

func (p *tcpRConnection) HandleFNF(f *FrameFNF) error {
	return nil
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

func (p *tcpRConnection) HandleRequestStream(f *FrameRequestStream) error {
	return nil
}

func (p *tcpRConnection) HandleRequestChannel(f *FrameRequestChannel) error {
	return nil
}

func (p *tcpRConnection) HandleRequestN(f *FrameRequestN) error {
	return nil
}

func (p *tcpRConnection) HandlePayload(f *FramePayload) error {
	return nil
}

func (p *tcpRConnection) HandleMetadataPush(f *FrameMetadataPush) error {
	return nil
}

func (p *tcpRConnection) HandleExtension(f *FrameExtension) error {
	return nil
}

func (p *tcpRConnection) rcvLoop(ctx context.Context) error {
	defer func() {
		if err := p.Close(); err != nil {
			log.Println("close connection failed:", err)
		}
	}()
	return p.decoder.Handle(ctx, func(frame Frame) (err error) {
		t := frame.Type()
		switch t {
		case SETUP:
			if p.handlerSetup != nil {
				err = p.handlerSetup(&FrameSetup{frame})
			}
		case LEASE:
			err = p.HandleLease(&FrameLease{frame})
		case KEEPALIVE:
			err = p.handleKeepalive(&FrameKeepalive{frame})
		case REQUEST_RESPONSE:
			err = p.HandleRequestResponse(&FrameRequestResponse{frame})
		case REQUEST_FNF:
			err = p.HandleFNF(&FrameFNF{frame})
		case REQUEST_STREAM:
			err = p.HandleRequestStream(&FrameRequestStream{frame})
		case REQUEST_CHANNEL:
			err = p.HandleRequestChannel(&FrameRequestChannel{frame})
		case REQUEST_N:
			err = p.HandleRequestN(&FrameRequestN{frame})
		case CANCEL:
			err = p.HandleCancel(&FrameCancel{frame})
		case PAYLOAD:
			err = p.HandlePayload(&FramePayload{frame})
		case ERROR:
			err = p.HandleError(&FrameError{frame})
		case METADATA_PUSH:
			err = p.HandleMetadataPush(&FrameMetadataPush{frame})
		//case RESUME:
		//case RESUME_OK:
		case EXT:
			err = p.HandleExtension(&FrameExtension{frame})
		default:
			return errUnknownFrame
		}
		return
	})
}

func (p *tcpRConnection) sndLoop(ctx context.Context) error {
	for frame := range p.snd {
		frameLength := writeUint24(len(frame))
		if _, err := p.c.Write(frameLength); err != nil {
			return err
		}
		if _, err := p.c.Write(frame); err != nil {
			return err
		}
	}
	return nil
}

func (p *tcpRConnection) handleKeepalive(f *FrameKeepalive) error {
	if !f.IsRespond() {
		return nil
	}
	return p.Send(NewFrameKeepalive(&BaseFrame{
		StreamID: 0,
		Type:     KEEPALIVE,
	}, false, f.LastReceivedPosition(), f.Payload()))
}

func newTcpRConnection(c net.Conn) *tcpRConnection {
	return &tcpRConnection{
		c:       c,
		decoder: newLengthBasedFrameDecoder(c),
		snd:     make(chan Frame, 64),
	}
}
