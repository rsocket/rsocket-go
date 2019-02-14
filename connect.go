package rsocket

import (
	"context"
	"io"
)

type RConnection interface {
	io.Closer
	RTransport
	Send(first Frame, others ...Frame) error
	PostFlight(ctx context.Context)
}
