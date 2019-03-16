package rsocket

import (
	"github.com/rsocket/rsocket-go/common"
	"github.com/rsocket/rsocket-go/framing"
	"time"
)

var defaultMimeType = []byte("application/binary")

// Payload is a stream message (upstream or downstream).
// It contains data associated with a stream created by a previous request.
// In Reactive Streams and Rx this is the 'onNext' event.
type Payload interface {
	// Metadata returns raw metadata bytes.
	Metadata() []byte
	// Data returns raw data bytes.
	Data() []byte
	// Release release all resources of payload.
	// Some payload implements is pooled, so you must release resoures after using it.
	Release()
}

// NewPayload create a new payload.
func NewPayload(data []byte, metadata []byte) Payload {
	return framing.NewFramePayload(0, data, metadata)
}

// NewPayloadString create a new payload with strings.
func NewPayloadString(data string, metadata string) Payload {
	return NewPayload([]byte(data), []byte(metadata))
}

// SetupPayload is particular payload for RSocket Setup.
type SetupPayload interface {
	Payload
	// DataMimeType returns MIME type of data.
	DataMimeType() string
	// MetadataMimeType returns MIME type of metadata.
	MetadataMimeType() string
	// TimeBetweenKeepalive returns interval duration of keepalive.
	TimeBetweenKeepalive() time.Duration
	// MaxLifetime returns max lifetime of RSocket connection.
	MaxLifetime() time.Duration
	// Version return RSocket protocol version.
	Version() common.Version
}

// ServerAcceptor is alias for server accepter.
type ServerAcceptor = func(setup SetupPayload, sendingSocket RSocket) RSocket

// RSocket is a contract providing different interaction models for RSocket protocol.
type RSocket interface {
	// FireAndForget is a single one-way message.
	FireAndForget(payload Payload)
	// MetadataPush sends asynchronous Metadata frame.
	MetadataPush(payload Payload)
	// RequestResponse request single response.
	RequestResponse(payload Payload) Mono
	// RequestStream request a completable stream.
	RequestStream(payload Payload) Flux
	// RequestChannel request a completable stream in both directions.
	RequestChannel(payloads Publisher) Flux
}

// NewAbstractSocket returns an abstract implementation of RSocket.
// You can specify the actual implementation of any request.
func NewAbstractSocket(opts ...OptAbstractSocket) RSocket {
	sk := &abstractRSocket{}
	for _, fn := range opts {
		fn(sk)
	}
	return sk
}

// OptAbstractSocket is option for abstract socket.
type OptAbstractSocket func(*abstractRSocket)

// MetadataPush register request handler for MetadataPush.
func MetadataPush(fn func(payload Payload)) OptAbstractSocket {
	return func(socket *abstractRSocket) {
		socket.metadataPush = fn
	}
}

// FireAndForget register request handler for FireAndForget.
func FireAndForget(fn func(payload Payload)) OptAbstractSocket {
	return func(opts *abstractRSocket) {
		opts.fireAndForget = fn
	}
}

// RequestResponse register request handler for RequestResponse.
func RequestResponse(fn func(payload Payload) Mono) OptAbstractSocket {
	return func(opts *abstractRSocket) {
		opts.requestResponse = fn
	}
}

// RequestStream register request handler for RequestStream.
func RequestStream(fn func(payload Payload) Flux) OptAbstractSocket {
	return func(opts *abstractRSocket) {
		opts.requestStream = fn
	}
}

// RequestChannel register request handler for RequestChannel.
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
