package rsocket

import "io"

type Transport interface {
	Connect() (conn RConnection, err error)
}

type ServerTransport interface {
	io.Closer
	Accept(acceptor func(setup *FrameSetup, conn RConnection) error)
	Listen(onReady ...func()) error
}
