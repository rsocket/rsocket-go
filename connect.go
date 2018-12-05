package rsocket

import (
	"context"
	"io"
)

type RConnection interface {
	io.Closer
	Send(first Frame, others ...Frame) error
	HandleSetup(callback func(setup *FrameSetup) (err error))
	HandleFNF(callback func(frame *FrameFNF) (err error))
	HandleRequestResponse(callback func(frame *FrameRequestResponse) (err error))
	HandleRequestStream(callback func(frame *FrameRequestStream) (err error))
	HandlePayload(callback func(frame *FramePayload) (err error))
	PostFlight(ctx context.Context)
}
