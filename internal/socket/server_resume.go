package socket

import (
	"context"

	"github.com/rsocket/rsocket-go/internal/transport"
)

type resumeServerSocket struct {
	*baseSocket
	token []byte
}

func (p *resumeServerSocket) Pause() bool {
	p.socket.clearTransport()
	return true
}

func (p *resumeServerSocket) SetResponder(responder Responder) {
	p.socket.SetResponder(responder)
}

func (p *resumeServerSocket) SetTransport(tp *transport.Transport) {
	p.socket.SetTransport(tp)
}

func (p *resumeServerSocket) Token() (token []byte, ok bool) {
	token, ok = p.token, true
	return
}

func (p *resumeServerSocket) Start(ctx context.Context) error {
	defer func() {
		_ = p.Close()
	}()
	return p.socket.loopWrite(ctx)
}

// NewServerResume creates a new server-side socket with resume support.
func NewServerResume(socket *DuplexRSocket, token []byte) ServerSocket {
	return &resumeServerSocket{
		baseSocket: newBaseSocket(socket),
		token:      token,
	}
}
