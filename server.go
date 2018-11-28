package rsocket

import (
	"context"
	"github.com/jjeffcaii/go-rsocket/protocol"
)

type RSocketServer struct {
	transport protocol.ServerTransport
	acceptor  Acceptor
}

func (p *RSocketServer) Start(ctx context.Context) error {
	return p.transport.Listen(ctx)
}

type SetupPayload struct {
}

type RSocket struct {
}

type Acceptor = func(setup *SetupPayload, sendingSocket *RSocket) (err error)

type RSocketServerBuilder interface {
	Transport(transport protocol.ServerTransport) RSocketServerBuilder
	Acceptor(acceptor Acceptor) RSocketServerBuilder
	Build() (*RSocketServer, error)
}

type serverBuilder struct {
	transport protocol.ServerTransport
	acceptor  Acceptor
}

func (p *serverBuilder) Transport(transport protocol.ServerTransport) RSocketServerBuilder {
	p.transport = transport
	return p
}

func (p *serverBuilder) Acceptor(acceptor Acceptor) RSocketServerBuilder {
	p.acceptor = acceptor
	return p
}

func (p *serverBuilder) Build() (*RSocketServer, error) {
	return &RSocketServer{
		transport: p.transport,
		acceptor:  p.acceptor,
	}, nil
}

func Builder() RSocketServerBuilder {
	return &serverBuilder{}
}
