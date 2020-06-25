package socket

import (
	"sync"
	"time"

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

type baseSocket struct {
	socket   *DuplexRSocket
	closers  []func(error)
	once     sync.Once
	reqLease *leaser
}

func (p *baseSocket) refreshLease(ttl time.Duration, n int64) {
	deadline := time.Now().Add(ttl)
	if p.reqLease == nil {
		p.reqLease = newLeaser(deadline, n)
	} else {
		p.reqLease.refresh(deadline, n)
	}
}

func (p *baseSocket) FireAndForget(message payload.Payload) {
	if err := p.reqLease.allow(); err != nil {
		logger.Warnf("request FireAndForget failed: %v\n", err)
	}
	p.socket.FireAndForget(message)
}

func (p *baseSocket) MetadataPush(message payload.Payload) {
	p.socket.MetadataPush(message)
}

func (p *baseSocket) RequestResponse(message payload.Payload) mono.Mono {
	if err := p.reqLease.allow(); err != nil {
		return mono.Error(err)
	}
	return p.socket.RequestResponse(message)
}

func (p *baseSocket) RequestStream(message payload.Payload) flux.Flux {
	if err := p.reqLease.allow(); err != nil {
		return flux.Error(err)
	}
	return p.socket.RequestStream(message)
}

func (p *baseSocket) RequestChannel(messages rx.Publisher) flux.Flux {
	if err := p.reqLease.allow(); err != nil {
		return flux.Error(err)
	}
	return p.socket.RequestChannel(messages)
}

func (p *baseSocket) OnClose(fn func(error)) {
	if fn != nil {
		p.closers = append(p.closers, fn)
	}
}

func (p *baseSocket) Close() (err error) {
	p.once.Do(func() {
		err = p.socket.Close()
		for i, l := 0, len(p.closers); i < l; i++ {
			func(fn func(error)) {
				defer func() {
					if e := tryRecover(recover()); e != nil {
						logger.Errorf("handle socket closer failed: %s\n", e)
					}
				}()
				fn(err)
			}(p.closers[l-i-1])
		}
	})
	return
}

func newBaseSocket(rawSocket *DuplexRSocket) *baseSocket {
	return &baseSocket{
		socket: rawSocket,
	}
}
