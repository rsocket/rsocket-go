package transport

import (
	"io"
	"time"

	"github.com/rsocket/rsocket-go/internal/framing"
)

// Conn is connection for RSocket.
type Conn interface {
	io.Closer
	// SetDeadline set deadline for current connection.
	// After this deadline, connection will be closed.
	SetDeadline(deadline time.Time) error
	// SetCounter bind a counter which can count r/w bytes.
	SetCounter(c *Counter)
	// Read reads next frame from Conn.
	Read() (framing.Frame, error)
	// Write writes a frame to Conn.
	Write(frames framing.Frame) error
	// Flush.
	Flush() error
}
