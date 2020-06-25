package socket

import (
	"context"

	"github.com/rsocket/rsocket-go/core/transport"
)

type serverSocket struct {
	*baseSocket
}

func (p *serverSocket) Pause() bool {
	return false
}

func (p *serverSocket) SetResponder(responder Responder) {
	p.socket.SetResponder(responder)
}

func (p *serverSocket) SetTransport(tp *transport.Transport) {
	p.socket.SetTransport(tp)
}

func (p *serverSocket) Token() (token []byte, ok bool) {
	return
}

func (p *serverSocket) Start(ctx context.Context) error {
	defer func() {
		_ = p.Close()
	}()
	return p.socket.loopWrite(ctx)
}

// NewServer creates a new server-side socket.
func NewServer(socket *DuplexRSocket) ServerSocket {
	return &serverSocket{
		baseSocket: newBaseSocket(socket),
	}
}
