package transport

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/rsocket/rsocket-go/core"
)

type (
	ClientTransporter func(context.Context) (*Transport, error)
	ServerTransporter func(context.Context) (ServerTransport, error)
)

type ListenerFactory func(context.Context) (net.Listener, error)

// Conn is connection for RSocket.
type Conn interface {
	io.Closer
	// SetDeadline set deadline for current connection.
	// After this deadline, connection will be closed.
	SetDeadline(deadline time.Time) error
	// SetCounter bind a counter which can count r/w bytes.
	SetCounter(c *core.TrafficCounter)
	// Read reads next frame from Conn.
	Read() (core.Frame, error)
	// Write writes a frame to Conn.
	Write(core.WriteableFrame) error
	// Flush.
	Flush() error
}
