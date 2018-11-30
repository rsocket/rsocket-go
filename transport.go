package rsocket

import "context"

type Transport interface {
	Connect(addr string) (conn RConnection, err error)
}

type ServerTransport interface {
	Accept(acceptor func(setup *FrameSetup, conn RConnection) error)
	Listen(ctx context.Context) error
}
