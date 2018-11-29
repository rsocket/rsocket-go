package protocol

import (
	"context"
	"github.com/pkg/errors"
	"log"
	"net"
	"time"
)

var errUnknownFrame = errors.New("bad frame")

type tcpRConnection struct {
	c       net.Conn
	decoder FrameDecoder
	snd     chan Frame

	hSetup  func(*FrameSetup) error
	hReqRes func(*FrameRequestResponse) error
}

func (p *tcpRConnection) HandleRequestResponse(callback func(frame *FrameRequestResponse) (err error)) {
	p.hReqRes = callback
}

func (p *tcpRConnection) HandleSetup(h func(setup *FrameSetup) (err error)) {
	p.hSetup = h
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
	log.Println("stream:", string(f.Payload()))
	for i := 0; i < 100; i++ {
		time.Sleep(10 * time.Millisecond)
		ts := time.Now().Format(string(f.Payload()))
		foo := NewPayload(&BaseFrame{
			Type:     PAYLOAD,
			Flags:    FlagNext,
			StreamID: f.StreamID(),
		}, []byte(ts), nil)
		if err := p.Send(foo); err != nil {
			return err
		}
	}
	foo := NewPayload(&BaseFrame{
		Type:     PAYLOAD,
		Flags:    FlagNext | FlagMetadata | FlagComplete,
		StreamID: f.StreamID(),
	}, []byte("abcde"), []byte("foobar"))
	return p.Send(foo)
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
			if p.hSetup != nil {
				err = p.hSetup(&FrameSetup{frame})
			}
		case LEASE:
			err = p.HandleLease(&FrameLease{frame})
		case KEEPALIVE:
			err = p.handleKeepalive(&FrameKeepalive{frame})
		case REQUEST_RESPONSE:
			err = p.hReqRes(&FrameRequestResponse{frame})
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
