package rsocket

import (
	"io"
	"time"
)

type Transport interface {
	Connect(keepaliveInterval time.Duration) (conn RConnection, err error)
}

type ServerTransport interface {
	io.Closer
	Accept(acceptor func(setup *frameSetup, conn RConnection) error)
	Listen(onReady ...func()) error
}
