package transport

import (
	"context"
	"io"

	"github.com/rsocket/rsocket-go/framing"
)

// FrameHandler is alias of frame handler.
type FrameHandler = func(frame framing.Frame) (err error)

// Transport is RSocket transport which is used to carry RSocket frames.
type Transport interface {
	io.Closer
	// Start start transport.
	Start(ctx context.Context) error
	// Send send a frame.
	Send(first framing.Frame) error
	// HandleSetup register Setup frame handler.
	HandleSetup(handler FrameHandler)
	// HandleFNF register FireAndForget frame handler.
	HandleFNF(handler FrameHandler)
	// HandleMetadataPush register MetadataPush frame handler.
	HandleMetadataPush(handler FrameHandler)
	// HandleRequestResponse register RequestResponse frame handler
	HandleRequestResponse(handler FrameHandler)
	// HandleRequestStream register RequestStream frame handler.
	HandleRequestStream(handler FrameHandler)
	// HandleRequestChannel register RequestChannel frame handler.
	HandleRequestChannel(handler FrameHandler)
	// HandlePayload register Payload frame handler.
	HandlePayload(handler FrameHandler)
	// HandleRequestN register RequestN frame handler.
	HandleRequestN(handler FrameHandler)
	// HandleError register Error frame handler.
	HandleError(handler FrameHandler)
	// HandleCancel register Cancel frame handler.
	HandleCancel(handler FrameHandler)
	// OnClose register close handler.
	OnClose(fn func())
}

// ServerTransport is server-side RSocket transport.
type ServerTransport interface {
	io.Closer
	// Accept register incoming connection handler.
	Accept(acceptor func(setup *framing.FrameSetup, conn Transport) error)
	// Listen listens on the network address addr and handles requests on incoming connections.
	// You can specify onReady handler, it'll be invoked when server begin listening.
	// It always returns a non-nil error.
	Listen(onReady ...func()) error
}
