package rsocket

import (
	"context"
	"github.com/jjeffcaii/go-rsocket/protocol"
)

type Server struct {
	transport protocol.ServerTransport
	acceptor  Acceptor
	handlerRQ HandlerRQ
}

func (p *Server) Start(ctx context.Context) error {
	p.transport.Accept(func(setup *protocol.FrameSetup, conn protocol.RConnection) error {
		rs := newRSocket(conn, p.handlerRQ)
		data := setup.Payload()
		metadata := setup.Metadata()
		sp := &SetupPayload{
			Payload: &Payload{
				Metadata: metadata,
				Data:     data,
			},
		}
		return p.acceptor(sp, rs)
	})
	return p.transport.Listen(ctx)
}

type Acceptor = func(setup *SetupPayload, sendingSocket *RSocket) (err error)
