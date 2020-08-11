package socket

import (
	"sync"
	"time"

	"github.com/rsocket/rsocket-go/logger"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/rsocket/rsocket-go/rx/mono"
)

type BaseSocket struct {
	socket   *DuplexConnection
	closers  []func(error)
	once     sync.Once
	reqLease *leaser
}

func (p *BaseSocket) refreshLease(ttl time.Duration, n int64) {
	deadline := time.Now().Add(ttl)
	if p.reqLease == nil {
		p.reqLease = newLeaser(deadline, n)
	} else {
		p.reqLease.refresh(deadline, n)
	}
}

func (p *BaseSocket) FireAndForget(message payload.Payload) {
	if err := p.reqLease.allow(); err != nil {
		logger.Warnf("request FireAndForget failed: %v\n", err)
	}
	p.socket.FireAndForget(message)
}

func (p *BaseSocket) MetadataPush(message payload.Payload) {
	p.socket.MetadataPush(message)
}

func (p *BaseSocket) RequestResponse(message payload.Payload) mono.Mono {
	if err := p.reqLease.allow(); err != nil {
		return mono.Error(err)
	}
	return p.socket.RequestResponse(message)
}

func (p *BaseSocket) RequestStream(message payload.Payload) flux.Flux {
	if err := p.reqLease.allow(); err != nil {
		return flux.Error(err)
	}
	return p.socket.RequestStream(message)
}

func (p *BaseSocket) RequestChannel(messages rx.Publisher) flux.Flux {
	if err := p.reqLease.allow(); err != nil {
		return flux.Error(err)
	}
	return p.socket.RequestChannel(messages)
}

func (p *BaseSocket) OnClose(fn func(error)) {
	if fn != nil {
		p.closers = append(p.closers, fn)
	}
}

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

func NewBaseSocket(rawSocket *DuplexConnection) *BaseSocket {
	return &BaseSocket{
		socket: rawSocket,
	}
}
