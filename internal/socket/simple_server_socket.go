package socket

import (
	"context"

	"github.com/rsocket/rsocket-go/core/transport"
)

type simpleServerSocket struct {
	*BaseSocket
}

func (p *simpleServerSocket) Pause() bool {
	return false
}

func (p *simpleServerSocket) SetResponder(responder Responder) {
	p.socket.SetResponder(responder)
}

func (p *simpleServerSocket) SetTransport(tp *transport.Transport) {
	p.socket.SetTransport(tp)
}

func (p *simpleServerSocket) Token() (token []byte, ok bool) {
	return
}

func (p *simpleServerSocket) Start(ctx context.Context) error {
	defer func() {
		_ = p.Close()
	}()
	return p.socket.LoopWrite(ctx)
}

// NewSimpleServerSocket creates a new server-side socket.
func NewSimpleServerSocket(socket *DuplexConnection) ServerSocket {
	return &simpleServerSocket{
		BaseSocket: NewBaseSocket(socket),
	}
}
