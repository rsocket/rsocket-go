package rsocket

import (
	"context"
	"fmt"
	"sync"
)

type clientTransportImpl struct {
	conn     rConnection
	handlers *sync.Map
	fnClose  func()
}

func (p *clientTransportImpl) Send(frame Frame) (err error) {
	err = p.conn.Send(frame)
	if err != nil {
		frame.Release()
	}
	return
}

func (p *clientTransportImpl) Close() error {
	return p.conn.Close()
}

func (p *clientTransportImpl) Start(ctx context.Context) error {
	p.conn.Handle(func(ctx context.Context, frame Frame) (err error) {
		t := frame.Header().Type()
		// 1. respond keepalive
		if t == tKeepalive {
			p.handleKeepalive(ctx, frame)
			return nil
		}
		// 2. skip invalid metadata push
		if t == tMetadataPush && frame.Header().StreamID() != 0 {
			frame.Release()
			logger.Warnf("rsocket.transport: omit MetadataPush with non-zero stream id %d\n", frame.Header().StreamID())
			return nil
		}
		// 3. trigger handler
		if h, ok := p.handlers.Load(t); ok {
			return h.(frameHandler)(frame)
		}
		// 4. missing handler
		frame.Release()
		return fmt.Errorf("missing frame handler: type=%s", t)
	})
	defer func() {
		if p.fnClose != nil {
			p.fnClose()
		}
	}()
	return p.conn.Start(ctx)
}

func (p *clientTransportImpl) onClose(fn func()) {
	p.fnClose = fn
}

func (p *clientTransportImpl) handleSetup(handler frameHandler) {
	p.handlers.Store(tSetup, handler)
}

func (p *clientTransportImpl) handleFNF(handler frameHandler) {
	p.handlers.Store(tRequestFNF, handler)
}

func (p *clientTransportImpl) handleMetadataPush(handler frameHandler) {
	p.handlers.Store(tMetadataPush, handler)
}

func (p *clientTransportImpl) handleRequestResponse(handler frameHandler) {
	p.handlers.Store(tRequestResponse, handler)
}

func (p *clientTransportImpl) handleRequestStream(handler frameHandler) {
	p.handlers.Store(tRequestStream, handler)
}

func (p *clientTransportImpl) handleRequestChannel(handler frameHandler) {
	p.handlers.Store(tRequestChannel, handler)
}

func (p *clientTransportImpl) handlePayload(handler frameHandler) {
	p.handlers.Store(tPayload, handler)
}

func (p *clientTransportImpl) handleRequestN(handler frameHandler) {
	p.handlers.Store(tRequestN, handler)
}

func (p *clientTransportImpl) handleError(handler frameHandler) {
	p.handlers.Store(tError, handler)
}

func (p *clientTransportImpl) handleCancel(handler frameHandler) {
	p.handlers.Store(tCancel, handler)
}

func (p *clientTransportImpl) handleKeepalive(ctx context.Context, frame Frame) {
	var err error
	select {
	case <-ctx.Done():
		err = ctx.Err()
	default:
		f := frame.(*frameKeepalive)
		if !f.header.Flag().Check(flagRespond) {
			f.Release()
		} else {
			f.header = createHeader(0, tKeepalive)
			err = p.conn.Send(f)
		}
	}
	if err != nil {
		logger.Errorf("handle keepalive failed: %s\n", err)
	}
}

func newTransportClient(c rConnection) *clientTransportImpl {
	return &clientTransportImpl{
		conn:     c,
		handlers: &sync.Map{},
	}
}
