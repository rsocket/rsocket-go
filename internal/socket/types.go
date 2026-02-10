package socket

import (
	"context"
	"io"
	"time"

	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/rsocket/rsocket-go/rx/mono"
)

// Closeable represents a closeable target.
type Closeable interface {
	io.Closer
	// OnClose bind a handler when closing.
	OnClose(closer func(error))
}

// Responder is a contract providing different interaction models for RSocket protocol.
type Responder interface {
	// FireAndForget is a single one-way message.
	FireAndForget(message payload.Payload)
	// MetadataPush sends asynchronous Metadata frame.
	MetadataPush(message payload.Payload)
	// RequestResponse request single response.
	RequestResponse(message payload.Payload) mono.Mono
	// RequestStream request a completable stream.
	RequestStream(message payload.Payload) flux.Flux
	// RequestChannel request a completable stream in both directions.
	RequestChannel(initialMessage payload.Payload, messages flux.Flux) flux.Flux
}

// ClientSocket represents a client-side socket.
type ClientSocket interface {
	Closeable
	Responder
	// Setup setups current socket.
	Setup(ctx context.Context, connectTimeout time.Duration, setup *SetupInfo) error
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
	// SetAddr sets address info.
	SetAddr(addr string)
}
