package socket

import (
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/logger"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/rsocket/rsocket-go/rx/mono"
)

var (
	errUnimplementedMetadataPush    = errors.New("METADATA_PUSH is unimplemented")
	errUnimplementedFireAndForget   = errors.New("FIRE_AND_FORGET is unimplemented")
	errUnimplementedRequestResponse = errors.New("REQUEST_RESPONSE is unimplemented")
	errUnimplementedRequestStream   = errors.New("REQUEST_STREAM is unimplemented")
	errUnimplementedRequestChannel  = errors.New("REQUEST_CHANNEL is unimplemented")
)

// AbstractRSocket represents an abstract RSocket.
type AbstractRSocket struct {
	FF func(payload.Payload)
	MP func(payload.Payload)
	RR func(payload.Payload) mono.Mono
	RS func(payload.Payload) flux.Flux
	RC func(rx.Publisher) flux.Flux
}

// MetadataPush starts a request of MetadataPush.
func (p AbstractRSocket) MetadataPush(message payload.Payload) {
	if p.MP == nil {
		logger.Errorf("%s\n", errUnimplementedMetadataPush)
		return
	}
	p.MP(message)
}

// FireAndForget starts a request of FireAndForget.
func (p AbstractRSocket) FireAndForget(message payload.Payload) {
	if p.FF == nil {
		logger.Errorf("%s\n", errUnimplementedFireAndForget)
		return
	}
	p.FF(message)
}

// RequestResponse starts a request of RequestResponse.
func (p AbstractRSocket) RequestResponse(message payload.Payload) mono.Mono {
	if p.RR == nil {
		return mono.Error(errUnimplementedRequestResponse)
	}
	return p.RR(message)
}

// RequestStream starts a request of RequestStream.
func (p AbstractRSocket) RequestStream(message payload.Payload) flux.Flux {
	if p.RS == nil {
		return flux.Error(errUnimplementedRequestStream)
	}
	return p.RS(message)
}

// RequestChannel starts a request of RequestChannel.
func (p AbstractRSocket) RequestChannel(messages rx.Publisher) flux.Flux {
	if p.RC == nil {
		return flux.Error(errUnimplementedRequestChannel)
	}
	return p.RC(messages)
}
