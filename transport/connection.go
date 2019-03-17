package transport

import (
	"context"
	"github.com/rsocket/rsocket-go/framing"
	"io"
)

type conn interface {
	io.Closer
	Handle(handler func(ctx context.Context, frame framing.Frame) error)
	Send(frame framing.Frame) error
	Write(frame framing.Frame) error
	Start(ctx context.Context) error
}
