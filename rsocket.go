package rsocket

import "errors"

var (
	MimeTypeBinary = []byte("application/binary")
	MimeTypeJSON   = []byte("application/json")
)

var (
	ErrInvalidTransport   = errors.New("rsocket: invalid transport")
	ErrInvalidFrame       = errors.New("rsocket: invalid frame")
	ErrInvalidContext     = errors.New("rsocket: invalid context")
	ErrInvalidFrameLength = errors.New("rsocket: invalid frame length")
	ErrReleasedResource   = errors.New("rsocket: resource has been released")
	ErrInvalidEmitter     = errors.New("rsocket: invalid emitter")
	ErrHandlerNil         = errors.New("rsocket: handler cannot be nil")
	ErrHandlerExist       = errors.New("rsocket: handler exists already")
	ErrSendFull           = errors.New("rsocket: frame send channel is full")
)

type ServerAcceptor = func(setup SetupPayload, sendingSocket RSocket) RSocket

type RSocket interface {
	FireAndForget(payload Payload)
	MetadataPush(payload Payload)
	RequestResponse(payload Payload) Mono
	RequestStream(payload Payload) Flux
	RequestChannel(payloads Publisher) Flux
}

func NewAbstractSocket(opts ...OptAbstractSocket) RSocket {
	sk := &abstractRSocket{}
	for _, fn := range opts {
		fn(sk)
	}
	return sk
}

type OptAbstractSocket func(*abstractRSocket)

func MetadataPush(fn func(payload Payload)) OptAbstractSocket {
	return func(socket *abstractRSocket) {
		socket.metadataPush = fn
	}
}

func FireAndForget(fn func(payload Payload)) OptAbstractSocket {
	return func(opts *abstractRSocket) {
		opts.fireAndForget = fn
	}
}

func RequestResponse(fn func(payload Payload) Mono) OptAbstractSocket {
	return func(opts *abstractRSocket) {
		opts.requestResponse = fn
	}
}

func RequestStream(fn func(payload Payload) Flux) OptAbstractSocket {
	return func(opts *abstractRSocket) {
		opts.requestStream = fn
	}
}

func RequestChannel(fn func(payloads Publisher) Flux) OptAbstractSocket {
	return func(opts *abstractRSocket) {
		opts.requestChannel = fn
	}
}

type abstractRSocket struct {
	fireAndForget   func(payload Payload)
	metadataPush    func(payload Payload)
	requestResponse func(payload Payload) Mono
	requestStream   func(payload Payload) Flux
	requestChannel  func(payloads Publisher) Flux
}

func (p *abstractRSocket) FireAndForget(payload Payload) {
	p.fireAndForget(payload)
}

func (p *abstractRSocket) RequestResponse(payload Payload) Mono {
	if p.requestResponse == nil {
		return nil
	}
	return p.requestResponse(payload)
}

func (p *abstractRSocket) RequestStream(payload Payload) Flux {
	return p.requestStream(payload)
}

func (p *abstractRSocket) RequestChannel(payloads Publisher) Flux {
	return p.requestChannel(payloads)
}

func (p *abstractRSocket) MetadataPush(payload Payload) {
	p.metadataPush(payload)
}
