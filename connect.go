package rsocket

import "io"

type RConnection interface {
	io.Closer
	Send(first Frame, others ...Frame) error
	HandleSetup(callback func(setup *FrameSetup) (err error))
	HandleRequestResponse(callback func(frame *FrameRequestResponse) (err error))
	HandleRequestStream(callback func(frame *FrameRequestStream) (err error))
}
