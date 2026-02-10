package rsocket

import (
	"context"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/socket"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/rsocket/rsocket-go/rx/mono"
)

const (
	// ErrorCodeInvalidSetup means the setup frame is invalid for the server.
	ErrorCodeInvalidSetup = core.ErrorCodeInvalidSetup
	// ErrorCodeUnsupportedSetup means some (or all) of the parameters specified by the client are unsupported by the server.
	ErrorCodeUnsupportedSetup = core.ErrorCodeUnsupportedSetup
	// ErrorCodeRejectedSetup means server rejected the setup, it can specify the reason in the payload.
	ErrorCodeRejectedSetup = core.ErrorCodeRejectedSetup
	// ErrorCodeRejectedResume means server rejected the resume, it can specify the reason in the payload.
	ErrorCodeRejectedResume = core.ErrorCodeRejectedResume
	// ErrorCodeConnectionError means the connection is being terminated.
	ErrorCodeConnectionError = core.ErrorCodeConnectionError
	// ErrorCodeConnectionClose means the connection is being terminated.
	ErrorCodeConnectionClose = core.ErrorCodeConnectionClose
	// ErrorCodeApplicationError means application layer logic generating a Reactive Streams onError event.
	ErrorCodeApplicationError = core.ErrorCodeApplicationError
	// ErrorCodeRejected means Responder reject it.
	ErrorCodeRejected = core.ErrorCodeRejected
	// ErrorCodeCanceled means the Responder canceled the request but may have started processing it (similar to REJECTED but doesn't guarantee lack of side-effects).
	ErrorCodeCanceled = core.ErrorCodeCanceled
	// ErrorCodeInvalid means the request is invalid.
	ErrorCodeInvalid = core.ErrorCodeInvalid
)

// Aliases for Error defines.
type (
	// ErrorCode is code for RSocket error.
	ErrorCode = core.ErrorCode
	// Error provides a method of accessing code and data.
	Error = core.CustomError
)

type (
	// ServerAcceptor is alias for server acceptor.
	ServerAcceptor = func(ctx context.Context, setup payload.SetupPayload, socket CloseableRSocket) (RSocket, error)

	// RSocket is a contract providing different interaction models for RSocket protocol.
	RSocket interface {
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

	// CloseableRSocket is RSocket which can be closed and handle close event.
	CloseableRSocket interface {
		socket.Closeable
		RSocket
	}

	// addressedRSocket is RSocket which contains address info.
	addressedRSocket interface {
		RSocket
		// Addr returns the address info.
		Addr() (string, bool)
	}

	// OptAbstractSocket is option for abstract socket.
	OptAbstractSocket func(*socket.AbstractRSocket)
)

// NewAbstractSocket returns an abstract implementation of RSocket.
// You can specify the actual implementation of any request.
func NewAbstractSocket(opts ...OptAbstractSocket) RSocket {
	sk := &socket.AbstractRSocket{}
	for _, fn := range opts {
		fn(sk)
	}
	return sk
}

// MetadataPush register request handler for MetadataPush.
func MetadataPush(fn func(request payload.Payload)) OptAbstractSocket {
	return func(socket *socket.AbstractRSocket) {
		socket.MP = fn
	}
}

// FireAndForget register request handler for FireAndForget.
func FireAndForget(fn func(request payload.Payload)) OptAbstractSocket {
	return func(opts *socket.AbstractRSocket) {
		opts.FF = fn
	}
}

// RequestResponse register request handler for RequestResponse.
func RequestResponse(fn func(request payload.Payload) (response mono.Mono)) OptAbstractSocket {
	return func(opts *socket.AbstractRSocket) {
		opts.RR = fn
	}
}

// RequestStream register request handler for RequestStream.
func RequestStream(fn func(request payload.Payload) (responses flux.Flux)) OptAbstractSocket {
	return func(opts *socket.AbstractRSocket) {
		opts.RS = fn
	}
}

// RequestChannel register request handler for RequestChannel.
func RequestChannel(fn func(initialRequest payload.Payload, requests flux.Flux) (responses flux.Flux)) OptAbstractSocket {
	return func(opts *socket.AbstractRSocket) {
		opts.RC = fn
	}
}

// GetAddr returns the address info of given RSocket.
// Normally, the format is "IP:PORT".
func GetAddr(rs RSocket) (string, bool) {
	if ars, ok := rs.(addressedRSocket); ok {
		return ars.Addr()
	}
	return "", false
}
