package socket

import (
	"context"
	"io"
	"sync"

	"github.com/rsocket/rsocket-go/internal/transport"
	. "github.com/rsocket/rsocket-go/payload"
	. "github.com/rsocket/rsocket-go/rx"
)

type Closeable interface {
	io.Closer
	OnClose(closer func())
}

// Responder is a contract providing different interaction models for RSocket protocol.
type Responder interface {
	// FireAndForget is a single one-way message.
	FireAndForget(msg Payload)
	// MetadataPush sends asynchronous Metadata frame.
	MetadataPush(msg Payload)
	// RequestResponse request single response.
	RequestResponse(msg Payload) Mono
	// RequestStream request a completable stream.
	RequestStream(msg Payload) Flux
	// RequestChannel request a completable stream in both directions.
	RequestChannel(msgs Publisher) Flux
}

type ClientSocket interface {
	Closeable
	Responder
	Setup(ctx context.Context, setup *SetupInfo) (err error)
}

type ServerSocket interface {
	Closeable
	Responder
	SetResponder(responder Responder)
	SetTransport(tp *transport.Transport)
	Pause() bool
	Start(ctx context.Context) error
	Token() (token []byte, ok bool)
}

type AbstractRSocket struct {
	FF func(Payload)
	MP func(Payload)
	RR func(Payload) Mono
	RS func(Payload) Flux
	RC func(Publisher) Flux
}

func (p AbstractRSocket) MetadataPush(msg Payload) {
	p.MP(msg)
}

func (p AbstractRSocket) FireAndForget(msg Payload) {
	p.FF(msg)
}

func (p AbstractRSocket) RequestResponse(msg Payload) Mono {
	return p.RR(msg)
}

func (p AbstractRSocket) RequestStream(msg Payload) Flux {
	return p.RS(msg)
}

func (p AbstractRSocket) RequestChannel(msgs Publisher) Flux {
	return p.RC(msgs)
}

type baseSocket struct {
	socket  *DuplexRSocket
	closers []func()
	once    *sync.Once
}

func (p *baseSocket) FireAndForget(msg Payload) {
	p.socket.FireAndForget(msg)
}

func (p *baseSocket) MetadataPush(msg Payload) {
	p.socket.MetadataPush(msg)
}

func (p *baseSocket) RequestResponse(msg Payload) Mono {
	return p.socket.RequestResponse(msg)
}

func (p *baseSocket) RequestStream(msg Payload) Flux {
	return p.socket.RequestStream(msg)
}

func (p *baseSocket) RequestChannel(msgs Publisher) Flux {
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
		once:   &sync.Once{},
	}
}
