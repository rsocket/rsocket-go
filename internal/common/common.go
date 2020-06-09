package common

import (
	"time"
)

const (
	// DefaultKeepaliveInterval is default keepalive interval duration.
	DefaultKeepaliveInterval = 20 * time.Second
	// DefaultKeepaliveMaxLifetime is default keepalive max lifetime.
	DefaultKeepaliveMaxLifetime = 90 * time.Second
)
