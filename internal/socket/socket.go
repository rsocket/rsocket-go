package socket

import (
	"context"
	"io"
	"sync"

	"github.com/rsocket/rsocket-go/internal/transport"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/rsocket/rsocket-go/rx/mono"
)

// Closeable represents a closeable target.
type Closeable interface {
	io.Closer
	// OnClose bind a handler when closing.
	OnClose(closer func())
}

// Responder is a contract providing different interaction models for RSocket protocol.
type Responder interface {
	// FireAndForget is a single one-way message.
	FireAndForget(msg payload.Payload)
	// MetadataPush sends asynchronous Metadata frame.
	MetadataPush(msg payload.Payload)
	// RequestResponse request single response.
	RequestResponse(msg payload.Payload) mono.Mono
	// RequestStream request a completable stream.
	RequestStream(msg payload.Payload) flux.Flux
	// RequestChannel request a completable stream in both directions.
	RequestChannel(msgs rx.Publisher) flux.Flux
}

// ClientSocket represents a client-side socket.
type ClientSocket interface {
	Closeable
	Responder
	// Setup setups current socket.
	Setup(ctx context.Context, setup *SetupInfo) (err error)
}

// ServerSocket represents a server-side socket.
type ServerSocket interface {
	Closeable
	Responder
	// SetResponder sets a responder for current socket.
	SetResponder(responder Responder)
	// SetTransport sets a transport for current socket.
	SetTransport(tp *transport.Transport)
	// Pause pause current socket.
	Pause() bool
	// Start starts current socket.
	Start(ctx context.Context) error
	// Token returns token of socket.
	Token() (token []byte, ok bool)
}

// AbstractRSocket represents an abstract RSocket.
type AbstractRSocket struct {
	FF func(payload.Payload)
	MP func(payload.Payload)
	RR func(payload.Payload) mono.Mono
	RS func(payload.Payload) flux.Flux
	RC func(rx.Publisher) flux.Flux
}

// MetadataPush starts a request of MetadataPush.
func (p AbstractRSocket) MetadataPush(msg payload.Payload) {
	p.MP(msg)
}

// FireAndForget starts a request of FireAndForget.
func (p AbstractRSocket) FireAndForget(msg payload.Payload) {
	p.FF(msg)
}

// RequestResponse starts a request of RequestResponse.
func (p AbstractRSocket) RequestResponse(msg payload.Payload) mono.Mono {
	return p.RR(msg)
}

// RequestStream starts a request of RequestStream.
func (p AbstractRSocket) RequestStream(msg payload.Payload) flux.Flux {
	return p.RS(msg)
}

// RequestChannel starts a request of RequestChannel.
func (p AbstractRSocket) RequestChannel(msgs rx.Publisher) flux.Flux {
	return p.RC(msgs)
}

type baseSocket struct {
	socket  *DuplexRSocket
	closers []func()
	once    sync.Once
}

func (p *baseSocket) FireAndForget(msg payload.Payload) {
	p.socket.FireAndForget(msg)
}

func (p *baseSocket) MetadataPush(msg payload.Payload) {
	p.socket.MetadataPush(msg)
}

func (p *baseSocket) RequestResponse(msg payload.Payload) mono.Mono {
	return p.socket.RequestResponse(msg)
}

func (p *baseSocket) RequestStream(msg payload.Payload) flux.Flux {
	return p.socket.RequestStream(msg)
}

func (p *baseSocket) RequestChannel(msgs rx.Publisher) flux.Flux {
	return p.socket.RequestChannel(msgs)
}

func (p *baseSocket) OnClose(fn func()) {
	if fn != nil {
		p.closers = append(p.closers, fn)
	}
}

func (p *baseSocket) Close() (err error) {
	p.once.Do(func() {
		err = p.socket.Close()
		for i, l := 0, len(p.closers); i < l; i++ {
			p.closers[l-i-1]()
		}
	})
	return
}

func newBaseSocket(rawSocket *DuplexRSocket) *baseSocket {
	return &baseSocket{
		socket: rawSocket,
	}
}
