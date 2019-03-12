package rsocket

import (
	"context"
	"io"
)

type rConnection interface {
	io.Closer
	Handle(handler func(ctx context.Context, frame Frame) error)
	Send(frame Frame) error
	Write(frame Frame) error
	Start(ctx context.Context) error
}
