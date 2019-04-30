package rsocket

import (
	"io"

	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

type (
	// ServerAcceptor is alias for server accepter.
	ServerAcceptor = func(setup payload.SetupPayload, sendingSocket EnhancedRSocket) RSocket

	// RSocket is a contract providing different interaction models for RSocket protocol.
	RSocket interface {
		// FireAndForget is a single one-way message.
		FireAndForget(msg payload.Payload)
		// MetadataPush sends asynchronous Metadata frame.
		MetadataPush(msg payload.Payload)
		// RequestResponse request single response.
		RequestResponse(msg payload.Payload) rx.Mono
		// RequestStream request a completable stream.
		RequestStream(msg payload.Payload) rx.Flux
		// RequestChannel request a completable stream in both directions.
		RequestChannel(msgs rx.Publisher) rx.Flux
	}

	// EnhancedRSocket is a RSocket which support more events.
	EnhancedRSocket interface {
		io.Closer
		RSocket
		// OnClose bind handler when socket disconnected.
		OnClose(fn func())
	}

	// OptAbstractSocket is option for abstract socket.
	OptAbstractSocket func(*abstractRSocket)
)

// NewAbstractSocket returns an abstract implementation of RSocket.
// You can specify the actual implementation of any request.
func NewAbstractSocket(opts ...OptAbstractSocket) RSocket {
	sk := &abstractRSocket{}
	for _, fn := range opts {
		fn(sk)
	}
	return sk
}

// MetadataPush register request handler for MetadataPush.
func MetadataPush(fn func(payload payload.Payload)) OptAbstractSocket {
	return func(socket *abstractRSocket) {
		socket.metadataPush = fn
	}
}

// FireAndForget register request handler for FireAndForget.
func FireAndForget(fn func(msg payload.Payload)) OptAbstractSocket {
	return func(opts *abstractRSocket) {
		opts.fireAndForget = fn
	}
}

// RequestResponse register request handler for RequestResponse.
func RequestResponse(fn func(msg payload.Payload) rx.Mono) OptAbstractSocket {
	return func(opts *abstractRSocket) {
		opts.requestResponse = fn
	}
}

// RequestStream register request handler for RequestStream.
func RequestStream(fn func(msg payload.Payload) rx.Flux) OptAbstractSocket {
	return func(opts *abstractRSocket) {
		opts.requestStream = fn
	}
}

// RequestChannel register request handler for RequestChannel.
func RequestChannel(fn func(msgs rx.Publisher) rx.Flux) OptAbstractSocket {
	return func(opts *abstractRSocket) {
		opts.requestChannel = fn
	}
}

type abstractRSocket struct {
	fireAndForget   func(payload.Payload)
	metadataPush    func(payload.Payload)
	requestResponse func(payload.Payload) rx.Mono
	requestStream   func(payload.Payload) rx.Flux
	requestChannel  func(rx.Publisher) rx.Flux
}

func (p *abstractRSocket) MetadataPush(msg payload.Payload) {
	p.metadataPush(msg)
}

func (p *abstractRSocket) FireAndForget(msg payload.Payload) {
	p.fireAndForget(msg)
}

func (p *abstractRSocket) RequestResponse(msg payload.Payload) rx.Mono {
	if p.requestResponse == nil {
		return nil
	}
	return p.requestResponse(msg)
}

func (p *abstractRSocket) RequestStream(msg payload.Payload) rx.Flux {
	return p.requestStream(msg)
}

func (p *abstractRSocket) RequestChannel(msgs rx.Publisher) rx.Flux {
	return p.requestChannel(msgs)
}
