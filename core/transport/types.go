package transport

import (
	"context"
	"io"
	"time"

	"github.com/rsocket/rsocket-go/core"
)

type (
	ClientTransportFunc = func(context.Context) (*Transport, error)
	ServerTransportFunc = func(context.Context) (ServerTransport, error)
)

// Conn is connection for RSocket.
type Conn interface {
	io.Closer
	// SetDeadline set deadline for current connection.
	// After this deadline, connection will be closed.
	SetDeadline(deadline time.Time) error
	// SetCounter bind a counter which can count r/w bytes.
	SetCounter(c *core.Counter)
	// Read reads next frame from Conn.
	Read() (core.Frame, error)
	// Write writes a frame to Conn.
	Write(core.FrameSupport) error
	// Flush.
	Flush() error
}
