package transport

import (
	"context"
	"io"

	"github.com/rsocket/rsocket-go/framing"
)

type conn interface {
	io.Closer
	Handle(handler func(ctx context.Context, frame framing.Frame) error)
	Send(frame framing.Frame) error
	Write(frame framing.Frame) error
	Start(ctx context.Context) error
}
