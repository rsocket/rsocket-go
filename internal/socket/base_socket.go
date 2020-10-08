package socket

import (
	"sync"
	"time"

	"github.com/rsocket/rsocket-go/logger"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/rsocket/rsocket-go/rx/mono"
)

// BaseSocket is basic socket.
type BaseSocket struct {
	socket   *DuplexConnection
	closers  []func(error)
	once     sync.Once
	reqLease *leaser
}

// FireAndForget sends FireAndForget request.
func (p *BaseSocket) FireAndForget(message payload.Payload) {
	if err := p.reqLease.allow(); err != nil {
		logger.Warnf("request FireAndForget failed: %v\n", err)
	}
	p.socket.FireAndForget(message)
}

// MetadataPush sends MetadataPush request.
func (p *BaseSocket) MetadataPush(message payload.Payload) {
	p.socket.MetadataPush(message)
}

// RequestResponse sends RequestResponse request.
func (p *BaseSocket) RequestResponse(message payload.Payload) mono.Mono {
	if err := p.reqLease.allow(); err != nil {
		return mono.Error(err)
	}
	return p.socket.RequestResponse(message)
}

// RequestStream sends RequestStream request.
func (p *BaseSocket) RequestStream(message payload.Payload) flux.Flux {
	if err := p.reqLease.allow(); err != nil {
		return flux.Error(err)
	}
	return p.socket.RequestStream(message)
}

// RequestChannel sends RequestChannel request.
func (p *BaseSocket) RequestChannel(messages flux.Flux) flux.Flux {
	if err := p.reqLease.allow(); err != nil {
		return flux.Error(err)
	}
	return p.socket.RequestChannel(messages)
}

// OnClose registers handler when socket closed.
func (p *BaseSocket) OnClose(fn func(error)) {
	if fn != nil {
		p.closers = append(p.closers, fn)
	}
}

// Close closes socket.
func (p *BaseSocket) Close() (err error) {
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

func (p *BaseSocket) refreshLease(ttl time.Duration, n int64) {
	deadline := time.Now().Add(ttl)
	if p.reqLease == nil {
		p.reqLease = newLeaser(deadline, n)
	} else {
		p.reqLease.refresh(deadline, n)
	}
}

// NewBaseSocket creates a new BaseSocket.
func NewBaseSocket(rawSocket *DuplexConnection) *BaseSocket {
	return &BaseSocket{
		socket: rawSocket,
	}
}
