package rsocket

import (
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/socket"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/rsocket/rsocket-go/rx/mono"
)

const (
	// ErrorCodeInvalidSetup means the setup frame is invalid for the server.
	ErrorCodeInvalidSetup = common.ErrorCodeInvalidSetup
	// ErrorCodeUnsupportedSetup means some (or all) of the parameters specified by the client are unsupported by the server.
	ErrorCodeUnsupportedSetup = common.ErrorCodeUnsupportedSetup
	// ErrorCodeRejectedSetup means server rejected the setup, it can specify the reason in the payload.
	ErrorCodeRejectedSetup = common.ErrorCodeRejectedSetup
	// ErrorCodeRejectedResume means server rejected the resume, it can specify the reason in the payload.
	ErrorCodeRejectedResume = common.ErrorCodeRejectedResume
	// ErrorCodeConnectionError means the connection is being terminated.
	ErrorCodeConnectionError = common.ErrorCodeConnectionError
	// ErrorCodeConnectionClose means the connection is being terminated.
	ErrorCodeConnectionClose = common.ErrorCodeConnectionClose
	// ErrorCodeApplicationError means application layer logic generating a Reactive Streams onError event.
	ErrorCodeApplicationError = common.ErrorCodeApplicationError
	// ErrorCodeRejected means Responder reject it.
	ErrorCodeRejected = common.ErrorCodeRejected
	// ErrorCodeCanceled means the Responder canceled the request but may have started processing it (similar to REJECTED but doesn't guarantee lack of side-effects).
	ErrorCodeCanceled = common.ErrorCodeCanceled
	// ErrorCodeInvalid means the request is invalid.
	ErrorCodeInvalid = common.ErrorCodeInvalid
)

type (
	// ErrorCode is code for RSocket error.
	ErrorCode = common.ErrorCode

	// Error provides a method of accessing code and data.
	Error interface {
		error
		// ErrorCode returns error code.
		ErrorCode() ErrorCode
		// ErrorData returns error data bytes.
		ErrorData() []byte
	}
)

type (
	// ServerAcceptor is alias for server acceptor.
	ServerAcceptor = func(setup payload.SetupPayload, sendingSocket CloseableRSocket) (RSocket, error)

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
		RequestChannel(messages rx.Publisher) flux.Flux
	}

	// CloseableRSocket is a RSocket which support more events.
	CloseableRSocket interface {
		socket.Closeable
		RSocket
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
func MetadataPush(fn func(msg payload.Payload)) OptAbstractSocket {
	return func(socket *socket.AbstractRSocket) {
		socket.MP = fn
	}
}

// FireAndForget register request handler for FireAndForget.
func FireAndForget(fn func(msg payload.Payload)) OptAbstractSocket {
	return func(opts *socket.AbstractRSocket) {
		opts.FF = fn
	}
}

// RequestResponse register request handler for RequestResponse.
func RequestResponse(fn func(msg payload.Payload) mono.Mono) OptAbstractSocket {
	return func(opts *socket.AbstractRSocket) {
		opts.RR = fn
	}
}

// RequestStream register request handler for RequestStream.
func RequestStream(fn func(msg payload.Payload) flux.Flux) OptAbstractSocket {
	return func(opts *socket.AbstractRSocket) {
		opts.RS = fn
	}
}

// RequestChannel register request handler for RequestChannel.
func RequestChannel(fn func(msgs rx.Publisher) flux.Flux) OptAbstractSocket {
	return func(opts *socket.AbstractRSocket) {
		opts.RC = fn
	}
}
