package socket

import (
	"sync"

	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

type baseSocket struct {
	socket  *DuplexRSocket
	closers []func()
	once    *sync.Once
}

func (p *baseSocket) FireAndForget(msg payload.Payload) {
	p.socket.FireAndForget(msg)
}

func (p *baseSocket) MetadataPush(msg payload.Payload) {
	p.socket.MetadataPush(msg)
}

func (p *baseSocket) RequestResponse(msg payload.Payload) rx.Mono {
	return p.socket.RequestResponse(msg)
}

func (p *baseSocket) RequestStream(msg payload.Payload) rx.Flux {
	return p.socket.RequestStream(msg)
}

func (p *baseSocket) RequestChannel(msgs rx.Publisher) rx.Flux {
	return p.socket.RequestChannel(msgs)
}

func (p *baseSocket) OnClose(fn func()) {
	if fn != nil {
		p.closers = append(p.closers, fn)
	}
}

func (p *baseSocket) Close() (err error) {
	p.once.Do(func() {
		err = p.socket.Close()
		for i, l := 0, len(p.closers); i < l; i++ {
			p.closers[l-i-1]()
		}
	})
	return
}

func newBaseSocket(rawSocket *DuplexRSocket) *baseSocket {
	return &baseSocket{
		socket: rawSocket,
		once:   &sync.Once{},
	}
}
