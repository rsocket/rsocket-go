package socket

import (
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/logger"
	"github.com/rsocket/rsocket-go/payload"
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
	RC func(payload.Payload, flux.Flux) flux.Flux
}

// MetadataPush starts a request of MetadataPush.
func (a AbstractRSocket) MetadataPush(message payload.Payload) {
	if a.MP == nil {
		logger.Errorf("%s\n", errUnimplementedMetadataPush)
		return
	}
	a.MP(message)
}

// FireAndForget starts a request of FireAndForget.
func (a AbstractRSocket) FireAndForget(message payload.Payload) {
	if a.FF == nil {
		logger.Errorf("%s\n", errUnimplementedFireAndForget)
		return
	}
	a.FF(message)
}

// RequestResponse starts a request of RequestResponse.
func (a AbstractRSocket) RequestResponse(message payload.Payload) mono.Mono {
	if a.RR == nil {
		return mono.Error(errUnimplementedRequestResponse)
	}
	return a.RR(message)
}

// RequestStream starts a request of RequestStream.
func (a AbstractRSocket) RequestStream(message payload.Payload) flux.Flux {
	if a.RS == nil {
		return flux.Error(errUnimplementedRequestStream)
	}
	return a.RS(message)
}

// RequestChannel starts a request of RequestChannel.
func (a AbstractRSocket) RequestChannel(initialRequest payload.Payload, messages flux.Flux) flux.Flux {
	if a.RC == nil {
		return flux.Error(errUnimplementedRequestChannel)
	}
	return a.RC(initialRequest, messages)
}
