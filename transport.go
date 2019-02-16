package rsocket

import (
	"context"
	"io"
)

type frameHandler = func(frame Frame) (err error)

type transport interface {
	io.Closer
	Start(ctx context.Context) error
	Send(first Frame) error
	handleSetup(handler frameHandler)
	handleFNF(handler frameHandler)
	handleMetadataPush(handler frameHandler)
	handleRequestResponse(handler frameHandler)
	handleRequestStream(handler frameHandler)
	handleRequestChannel(handler frameHandler)
	handlePayload(handler frameHandler)
	handleRequestN(handler frameHandler)
	handleError(handler frameHandler)
	handleCancel(handler frameHandler)
	onClose(fn func())
}

type ServerTransport interface {
	io.Closer
	Accept(acceptor func(setup *frameSetup, conn transport) error)
	Listen(onReady ...func()) error
}
