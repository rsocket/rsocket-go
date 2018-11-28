package protocol

import "context"

type Transport interface {
}

type ServerTransport interface {
	Transport
	Accept(acceptor func(setup *FrameSetup, conn RConnection) error)
	Listen(ctx context.Context) error
}
