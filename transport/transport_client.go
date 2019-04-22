package transport

import (
	"context"
	"fmt"
	"sync"

	"github.com/rsocket/rsocket-go/common/logger"
	"github.com/rsocket/rsocket-go/framing"
)

type clientTransportImpl struct {
	conn     conn
	handlers *sync.Map
	fnClose  []func()
}

func (p *clientTransportImpl) Send(frame framing.Frame) (err error) {
	err = p.conn.Send(frame)
	if err != nil {
		frame.Release()
	}
	return
}

func (p *clientTransportImpl) Close() error {
	return p.conn.Close()
}

func (p *clientTransportImpl) onFrame(ctx context.Context, frame framing.Frame) (err error) {
	header := frame.Header()
	typo := header.Type()
	// respond keepalive
	if typo == framing.FrameTypeKeepalive {
		p.HandleKeepalive(ctx, frame)
		return nil
	}
	// skip invalid metadata push
	if typo == framing.FrameTypeMetadataPush && header.StreamID() != 0 {
		frame.Release()
		logger.Warnf("rsocket.Transport: omit MetadataPush with non-zero stream id %d\n", header.StreamID())
		return nil
	}

	// trigger handler
	if h, ok := p.handlers.Load(typo); ok {
		return h.(FrameHandler)(frame)
	}
	// missing handler
	frame.Release()
	return fmt.Errorf("missing frame handler: type=%s", typo)
}

func (p *clientTransportImpl) Start(ctx context.Context) error {
	p.conn.Handle(p.onFrame)
	defer func() {
		for _, fn := range p.fnClose {
			fn()
		}
	}()
	return p.conn.Start(ctx)
}

func (p *clientTransportImpl) OnClose(fn func()) {
	p.fnClose = append(p.fnClose, fn)
}

func (p *clientTransportImpl) HandleSetup(handler FrameHandler) {
	p.handlers.Store(framing.FrameTypeSetup, handler)
}

func (p *clientTransportImpl) HandleFNF(handler FrameHandler) {
	p.handlers.Store(framing.FrameTypeRequestFNF, handler)
}

func (p *clientTransportImpl) HandleMetadataPush(handler FrameHandler) {
	p.handlers.Store(framing.FrameTypeMetadataPush, handler)
}

func (p *clientTransportImpl) HandleRequestResponse(handler FrameHandler) {
	p.handlers.Store(framing.FrameTypeRequestResponse, handler)
}

func (p *clientTransportImpl) HandleRequestStream(handler FrameHandler) {
	p.handlers.Store(framing.FrameTypeRequestStream, handler)
}

func (p *clientTransportImpl) HandleRequestChannel(handler FrameHandler) {
	p.handlers.Store(framing.FrameTypeRequestChannel, handler)
}

func (p *clientTransportImpl) HandlePayload(handler FrameHandler) {
	p.handlers.Store(framing.FrameTypePayload, handler)
}

func (p *clientTransportImpl) HandleRequestN(handler FrameHandler) {
	p.handlers.Store(framing.FrameTypeRequestN, handler)
}

func (p *clientTransportImpl) HandleError(handler FrameHandler) {
	p.handlers.Store(framing.FrameTypeError, handler)
}

func (p *clientTransportImpl) HandleCancel(handler FrameHandler) {
	p.handlers.Store(framing.FrameTypeCancel, handler)
}

func (p *clientTransportImpl) HandleKeepalive(ctx context.Context, frame framing.Frame) {
	var err error
	select {
	case <-ctx.Done():
		err = ctx.Err()
	default:
		f := frame.(*framing.FrameKeepalive)
		if !f.Header().Flag().Check(framing.FlagRespond) {
			f.Release()
		} else {
			f.SetHeader(framing.NewFrameHeader(0, framing.FrameTypeKeepalive))
			err = p.conn.Send(f)
		}
	}
	if err != nil {
		logger.Errorf("handle keepalive failed: %s\n", err)
	}
}

func newTransportClient(c conn) *clientTransportImpl {
	return &clientTransportImpl{
		conn:     c,
		handlers: &sync.Map{},
		fnClose:  make([]func(), 0),
	}
}
