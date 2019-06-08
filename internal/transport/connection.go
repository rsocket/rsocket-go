package transport

import (
	"context"
	"io"
	"time"

	. "github.com/rsocket/rsocket-go/internal/framing"
)

// Conn is connection for RSocket.
type Conn interface {
	io.Closer
	// SetDeadline set deadline for current connection.
	// After this deadline, connection will be closed.
	SetDeadline(deadline time.Time) error
	// SetCounter bind a counter which can count r/w bytes.
	SetCounter(c *Counter)
	// Handle bind a handler for incoming frames.
	Handle(handler func(ctx context.Context, frame Frame) error)
	// Write write a frame.
	Write(frame Frame) error
	// Start fire current connection.
	// Invoke this method will block current goroutine until connection disconnected.
	Start(ctx context.Context) (err error)
}
