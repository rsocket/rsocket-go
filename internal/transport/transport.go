package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/rsocket/rsocket-go/internal/common"
	. "github.com/rsocket/rsocket-go/internal/framing"
	"github.com/rsocket/rsocket-go/internal/logger"
)

// FrameHandler is alias of frame handler.
type FrameHandler = func(frame Frame) (err error)
type ServerTransportAcceptor = func(ctx context.Context, tp *Transport)

// ServerTransport is server-side RSocket transport.
type ServerTransport interface {
	io.Closer
	// Accept register incoming connection handler.
	Accept(acceptor ServerTransportAcceptor)
	// Listen listens on the network address addr and handles requests on incoming connections.
	// You can specify onReady handler, it'll be invoked when server begin listening.
	// It always returns a non-nil error.
	Listen(onReady ...func()) error
}

// Transport is RSocket transport which is used to carry RSocket frames.
type Transport struct {
	conn        Conn
	maxLifetime time.Duration
	lastRcvPos  uint64
	once        *sync.Once

	hSetup           FrameHandler
	hResume          FrameHandler
	hResumeOK        FrameHandler
	hFireAndForget   FrameHandler
	hMetadataPush    FrameHandler
	hRequestResponse FrameHandler
	hRequestStream   FrameHandler
	hRequestChannel  FrameHandler
	hPayload         FrameHandler
	hRequestN        FrameHandler
	hError           FrameHandler
	hError0          FrameHandler
	hCancel          FrameHandler
	hKeepalive       FrameHandler
}

func (p *Transport) HandleError0(handler FrameHandler) {
	p.hError0 = handler
}

// Connection returns current connection.
func (p *Transport) Connection() Conn {
	return p.conn
}

// SetLifetime set max lifetime for current transport.
func (p *Transport) SetLifetime(lifetime time.Duration) {
	if lifetime < 1 {
		return
	}
	p.maxLifetime = lifetime
}

// Send send a frame.
func (p *Transport) Send(frame Frame) (err error) {
	err = p.conn.Write(frame)
	return
}

func (p *Transport) Close() (err error) {
	p.once.Do(func() {
		err = p.conn.Close()
	})
	return
}

// Start start transport.
func (p *Transport) Start(ctx context.Context) (err error) {
	defer func() {
		if err := p.Close(); err != nil {
			logger.Warnf("close transport failed: %s\n", err)
		}
	}()
L:
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break L
		default:
			f, err := p.conn.Read()
			if err != nil {
				break L
			}
			err = p.onFrame(ctx, f)
			if err != nil {
				break L
			}
		}
	}
	return
}

func (p *Transport) HandleSetup(handler FrameHandler) {
	p.hSetup = handler
}

func (p *Transport) HandleResume(handler FrameHandler) {
	p.hResume = handler
}

func (p *Transport) HandleResumeOK(handler FrameHandler) {
	p.hResumeOK = handler
}

func (p *Transport) HandleFNF(handler FrameHandler) {
	p.hFireAndForget = handler
}

func (p *Transport) HandleMetadataPush(handler FrameHandler) {
	p.hMetadataPush = handler
}

func (p *Transport) HandleRequestResponse(handler FrameHandler) {
	p.hRequestResponse = handler
}

func (p *Transport) HandleRequestStream(handler FrameHandler) {
	p.hRequestStream = handler
}

func (p *Transport) HandleRequestChannel(handler FrameHandler) {
	p.hRequestChannel = handler
}

func (p *Transport) HandlePayload(handler FrameHandler) {
	p.hPayload = handler
}

func (p *Transport) HandleRequestN(handler FrameHandler) {
	p.hRequestN = handler
}

func (p *Transport) HandleError(handler FrameHandler) {
	p.hError = handler
}

func (p *Transport) HandleCancel(handler FrameHandler) {
	p.hCancel = handler
}

func (p *Transport) HandleKeepalive(handler FrameHandler) {
	p.hKeepalive = handler
}

func (p *Transport) onFrame(ctx context.Context, frame Frame) (err error) {
	header := frame.Header()
	t := header.Type()
	sid := header.StreamID()

	var handler FrameHandler

	switch t {
	case FrameTypeSetup:
		p.maxLifetime = frame.(*FrameSetup).MaxLifetime()
		handler = p.hSetup
	case FrameTypeResume:
		handler = p.hResume
	case FrameTypeResumeOK:
		p.lastRcvPos = frame.(*FrameResumeOK).LastReceivedClientPosition()
		handler = p.hResumeOK
	case FrameTypeRequestFNF:
		handler = p.hFireAndForget
	case FrameTypeMetadataPush:
		if sid != 0 {
			// skip invalid metadata push
			frame.Release()
			logger.Warnf("rsocket.Transport: omit MetadataPush with non-zero stream id %d\n", sid)
			return
		}
		handler = p.hMetadataPush
	case FrameTypeRequestResponse:
		handler = p.hRequestResponse
	case FrameTypeRequestStream:
		handler = p.hRequestStream
	case FrameTypeRequestChannel:
		handler = p.hRequestChannel
	case FrameTypePayload:
		handler = p.hPayload
	case FrameTypeRequestN:
		handler = p.hRequestN
	case FrameTypeError:
		if sid == 0 {
			err = errors.New(frame.(*FrameError).Error())
			if p.hError0 != nil {
				_ = p.hError0(frame)
			} else {
				frame.Release()
			}
			return
		}
		handler = p.hError
	case FrameTypeCancel:
		handler = p.hCancel
	case FrameTypeKeepalive:
		ka := frame.(*FrameKeepalive)
		p.lastRcvPos = ka.LastReceivedPosition()
		handler = p.hKeepalive
	}

	// Set deadline.
	deadline := time.Now().Add(p.maxLifetime)
	err = p.conn.SetDeadline(deadline)
	if err != nil {
		return
	}

	// missing handler
	if handler == nil {
		err = fmt.Errorf("missing frame handler: type=%s", t)
		frame.Release()
		return
	}

	// trigger handler
	err = handler(frame)
	return
}

func newTransportClient(c Conn) *Transport {
	return &Transport{
		conn:        c,
		maxLifetime: common.DefaultKeepaliveMaxLifetime,
		once:        &sync.Once{},
	}
}
