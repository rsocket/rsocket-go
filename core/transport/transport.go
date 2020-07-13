package transport

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/logger"
)

var errTransportClosed = errors.New("transport closed")

// FrameHandler is an alias of frame handler.
type FrameHandler = func(frame core.Frame) (err error)

// ServerTransportAcceptor is an alias of server transport handler.
type ServerTransportAcceptor = func(ctx context.Context, tp *Transport, onClose func(*Transport))

// ServerTransport is server-side RSocket transport.
type ServerTransport interface {
	io.Closer
	// Accept register incoming connection handler.
	Accept(acceptor ServerTransportAcceptor)
	// Listen listens on the network address addr and handles requests on incoming connections.
	// You can specify onReady handler, it'll be invoked when server begin listening.
	// It always returns a non-nil error.
	Listen(ctx context.Context, notifier chan<- struct{}) error
}

type EventType int

const (
	OnSetup EventType = iota
	OnResume
	OnLease
	OnResumeOK
	OnFireAndForget
	OnMetadataPush
	OnRequestResponse
	OnRequestStream
	OnRequestChannel
	OnPayload
	OnRequestN
	OnError
	OnErrorWithZeroStreamID
	OnCancel
	OnKeepalive

	handlerLen = int(OnKeepalive) + 1
)

// Transport is RSocket transport which is used to carry RSocket frames.
type Transport struct {
	conn        core.Conn
	maxLifetime time.Duration
	lastRcvPos  uint64
	once        sync.Once
	handlers    [handlerLen]FrameHandler
}

func (p *Transport) RegisterHandler(event EventType, handler FrameHandler) {
	p.handlers[int(event)] = handler
}

// Connection returns current connection.
func (p *Transport) Connection() core.Conn {
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
func (p *Transport) Send(frame core.FrameSupport, flush bool) (err error) {
	defer func() {
		// ensure frame done when send success.
		if err == nil {
			frame.Done()
		}
	}()
	if p == nil || p.conn == nil {
		err = errTransportClosed
		return
	}
	err = p.conn.Write(frame)
	if err != nil {
		return
	}
	if !flush {
		return
	}
	err = p.conn.Flush()
	return
}

// Flush flush all bytes in current connection.
func (p *Transport) Flush() (err error) {
	if p == nil || p.conn == nil {
		err = errTransportClosed
		return
	}
	err = p.conn.Flush()
	return
}

// Close close current transport.
func (p *Transport) Close() (err error) {
	p.once.Do(func() {
		err = p.conn.Close()
	})
	return
}

// ReadFirst reads first frame.
func (p *Transport) ReadFirst(ctx context.Context) (frame core.Frame, err error) {
	select {
	case <-ctx.Done():
		err = ctx.Err()
	default:
		frame, err = p.conn.Read()
		if err != nil {
			err = errors.Wrap(err, "read first frame failed")
		}
	}
	if err != nil {
		_ = p.Close()
	}
	return
}

// Start start transport.
func (p *Transport) Start(ctx context.Context) (err error) {
	defer p.Close()
L:
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
			f, err := p.conn.Read()
			if err != nil {
				break L
			}
			err = p.DispatchFrame(ctx, f)
			if err != nil {
				break L
			}
		}
	}
	if err == io.EOF {
		err = nil
		return
	}
	if err != nil {
		err = errors.Wrap(err, "read and delivery frame failed")
	}
	return
}

// DispatchFrame delivery incoming frames.
func (p *Transport) DispatchFrame(_ context.Context, frame core.Frame) (err error) {
	header := frame.Header()
	t := header.Type()
	sid := header.StreamID()

	var handler FrameHandler

	switch t {
	case core.FrameTypeSetup:
		p.maxLifetime = frame.(*framing.SetupFrame).MaxLifetime()
		handler = p.handlers[OnSetup]
	case core.FrameTypeResume:
		handler = p.handlers[OnResume]
	case core.FrameTypeResumeOK:
		p.lastRcvPos = frame.(*framing.ResumeOKFrame).LastReceivedClientPosition()
		handler = p.handlers[OnResumeOK]
	case core.FrameTypeRequestFNF:
		handler = p.handlers[OnFireAndForget]
	case core.FrameTypeMetadataPush:
		if sid != 0 {
			// skip invalid metadata push
			logger.Warnf("rsocket.Transport: omit MetadataPush with non-zero stream id %d\n", sid)
			return
		}
		handler = p.handlers[OnMetadataPush]
	case core.FrameTypeRequestResponse:
		handler = p.handlers[OnRequestResponse]
	case core.FrameTypeRequestStream:
		handler = p.handlers[OnRequestStream]
	case core.FrameTypeRequestChannel:
		handler = p.handlers[OnRequestChannel]
	case core.FrameTypePayload:
		handler = p.handlers[OnPayload]
	case core.FrameTypeRequestN:
		handler = p.handlers[OnRequestN]
	case core.FrameTypeError:
		if sid == 0 {
			err = errors.New(frame.(*framing.ErrorFrame).Error())
			if call := p.handlers[OnErrorWithZeroStreamID]; call != nil {
				_ = call(frame)
			}
			return
		}
		handler = p.handlers[OnError]
	case core.FrameTypeCancel:
		handler = p.handlers[OnCancel]
	case core.FrameTypeKeepalive:
		ka := frame.(*framing.KeepaliveFrame)
		p.lastRcvPos = ka.LastReceivedPosition()
		handler = p.handlers[OnKeepalive]
	case core.FrameTypeLease:
		handler = p.handlers[OnLease]
	}

	// Set deadline.
	deadline := time.Now().Add(p.maxLifetime)
	err = p.conn.SetDeadline(deadline)
	if err != nil {
		return
	}

	// missing handler
	if handler == nil {
		err = errors.Errorf("missing frame handler: type=%s", t)
		return
	}

	// trigger handler
	err = handler(frame)
	if err != nil {
		err = errors.Wrap(err, "exec frame handler failed")
	}
	return
}

func NewTransport(c core.Conn) *Transport {
	return &Transport{
		conn:        c,
		maxLifetime: common.DefaultKeepaliveMaxLifetime,
	}
}
