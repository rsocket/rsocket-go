package transport

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/internal/buffer"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/logger"
)

var (
	errTransportClosed = errors.New("transport closed")
	errNoHandler       = errors.New("you must register a handler")
)

// FrameHandler is an alias of frame handler.
type FrameHandler = func(frame core.BufferedFrame) (err error)

// ServerTransportAcceptor is an alias of server transport handler.
type ServerTransportAcceptor = func(ctx context.Context, tp *Transport, onClose func(*Transport))

// ServerTransport is server-side RSocket transport.
type ServerTransport interface {
	io.Closer
	// Accept register incoming connection handler.
	Accept(acceptor ServerTransportAcceptor)
	// Listen listens on the network address addr and handles requests on incoming connections.
	// You can specify notifier chan, it'll be sent true/false when server listening success/failed.
	Listen(ctx context.Context, notifier chan<- bool) error
}

// EventType represents the events when transport received frames.
type EventType int

// EventTypes
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
	mu          sync.RWMutex
	conn        Conn
	maxLifetime time.Duration
	lastRcvPos  uint64
	once        sync.Once
	handlers    [handlerLen]FrameHandler
}

// NewTransport creates new transport.
func NewTransport(c Conn) *Transport {
	return &Transport{
		conn:        c,
		maxLifetime: common.DefaultKeepaliveMaxLifetime,
	}
}

// IsNoHandlerError returns true if input error means no handler registered.
func IsNoHandlerError(err error) bool {
	return err == errNoHandler
}

func (p *Transport) Addr() (string, bool) {
	ac, ok := p.conn.(AddrConn)
	if ok {
		return ac.Addr(), true
	}
	return "", false
}

// Handle register event handlers
func (p *Transport) Handle(event EventType, handler FrameHandler) {
	p.mu.Lock()
	p.handlers[event] = handler
	p.mu.Unlock()
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
func (p *Transport) Send(frame core.WriteableFrame, flush bool) (err error) {
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
func (p *Transport) ReadFirst(ctx context.Context) (frame core.BufferedFrame, err error) {
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

func (p *Transport) loopReadBuffer(ctx context.Context, bf *buffer.Unbounded, errCh chan<- error, done chan<- struct{}) {
	defer func() {
		done <- struct{}{}
	}()
	c := bf.Get()
	for next := range c {
		err := p.DispatchFrame(ctx, next.(core.BufferedFrame))
		if err != nil {
			select {
			case errCh <- err:
			default:
			}
		}
		if bf.Load() == 0 {
			runtime.Gosched()
		}
	}
}

// Start start transport.
func (p *Transport) Start(ctx context.Context) error {
	defer func() {
		_ = p.Close()
	}()

	done := make(chan struct{}, 1)
	errChan := make(chan error, 1)

	var (
		oddFrames  = buffer.NewUnbounded()
		evenFrames = buffer.NewUnbounded()
	)

	defer func() {
		oddFrames.Dispose()
		evenFrames.Dispose()

		<-done
		<-done
	}()

	go p.loopReadBuffer(ctx, oddFrames, errChan, done)
	go p.loopReadBuffer(ctx, evenFrames, errChan, done)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errChan:
			return errors.Wrap(err, "dispatch incoming frame failed:")
		default:
			f, err := p.conn.Read()
			if err == io.EOF {
				return nil
			}
			if err != nil {
				return err
			}

			sid := f.StreamID()

			// put frame into buffer
			if sid&1 == 1 {
				oddFrames.Put(f)
				continue
			}
			if sid != 0 {
				evenFrames.Put(f)
				continue
			}

			// dispatch frame with zero stream id
			if err := p.DispatchFrame(ctx, f); err != nil {
				return errors.Wrap(err, "dispatch incoming frame failed:")
			}
		}
	}
}

// DispatchFrame delivery incoming frames.
func (p *Transport) DispatchFrame(_ context.Context, frame core.BufferedFrame) (err error) {
	header := frame.Header()
	t := header.Type()
	sid := header.StreamID()

	var handler FrameHandler

	switch t {
	case core.FrameTypeSetup:
		p.maxLifetime = frame.(*framing.SetupFrame).MaxLifetime()
		handler = p.getHandler(OnSetup)
	case core.FrameTypeResume:
		handler = p.getHandler(OnResume)
	case core.FrameTypeResumeOK:
		p.lastRcvPos = frame.(*framing.ResumeOKFrame).LastReceivedClientPosition()
		handler = p.getHandler(OnResumeOK)
	case core.FrameTypeRequestFNF:
		handler = p.getHandler(OnFireAndForget)
	case core.FrameTypeMetadataPush:
		if sid != 0 {
			// skip invalid metadata push
			logger.Warnf("rsocket: omit MetadataPush with non-zero stream id %d\n", sid)
			return
		}
		handler = p.getHandler(OnMetadataPush)
	case core.FrameTypeRequestResponse:
		handler = p.getHandler(OnRequestResponse)
	case core.FrameTypeRequestStream:
		handler = p.getHandler(OnRequestStream)
	case core.FrameTypeRequestChannel:
		handler = p.getHandler(OnRequestChannel)
	case core.FrameTypePayload:
		handler = p.getHandler(OnPayload)
	case core.FrameTypeRequestN:
		handler = p.getHandler(OnRequestN)
	case core.FrameTypeError:
		if sid == 0 {
			err = frame.(*framing.ErrorFrame).ToError()
			if call := p.getHandler(OnErrorWithZeroStreamID); call != nil {
				_ = call(frame)
			}
			return
		}
		handler = p.getHandler(OnError)
	case core.FrameTypeCancel:
		handler = p.getHandler(OnCancel)
	case core.FrameTypeKeepalive:
		ka := frame.(*framing.KeepaliveFrame)
		p.lastRcvPos = ka.LastReceivedPosition()
		handler = p.getHandler(OnKeepalive)
	case core.FrameTypeLease:
		handler = p.getHandler(OnLease)
	}

	// Set deadline.
	deadline := time.Now().Add(p.maxLifetime)
	err = p.conn.SetDeadline(deadline)
	if err != nil {
		return
	}

	// missing handler
	if handler == nil {
		err = errNoHandler
		return
	}

	// trigger handler
	err = handler(frame)
	if err != nil {
		err = errors.Wrap(err, fmt.Sprintf("handle frame %s failed:", frame.Header().Type()))
	}
	return
}

func (p *Transport) getHandler(t EventType) FrameHandler {
	p.mu.RLock()
	h := p.handlers[t]
	p.mu.RUnlock()
	return h
}
