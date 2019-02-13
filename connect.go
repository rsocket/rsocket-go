package rsocket

import (
	"context"
	"io"
)

type RConnection interface {
	io.Closer
	Send(first Frame, others ...Frame) error
	PostFlight(ctx context.Context)
	HandleSetup(callback func(setup *frameSetup) (err error))
	HandleFNF(callback func(f *frameFNF) (err error))
	HandleMetadataPush(callback func(f *frameMetadataPush) (err error))
	HandleRequestResponse(callback func(f *frameRequestResponse) (err error))
	HandleRequestStream(callback func(f *frameRequestStream) (err error))
	HandleRequestChannel(callback func(f *frameRequestChannel) (err error))
	HandlePayload(callback func(f *framePayload) (err error))
}
