package common

import (
	"time"
)

const (
	// DefaultKeepaliveInteval is default keepalive interval duration.
	DefaultKeepaliveInteval = 20 * time.Second
	// DefaultKeepaliveMaxLifetime is default keepalive max lifetime.
	DefaultKeepaliveMaxLifetime = 90 * time.Second
)

