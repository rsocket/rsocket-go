package common

import (
	"fmt"
	"time"
)

const (
	// DefaultKeepaliveInteval is default keepalive interval duration.
	DefaultKeepaliveInteval = 20 * time.Second
	// DefaultKeepaliveMaxLifetime is default keepalive max lifetime.
	DefaultKeepaliveMaxLifetime = 90 * time.Second
)

// Some TCP connection settings.
var (
	DefaultTCPWriteBuffSize = 16 * 1024
	DefaultTCPReadBuffSize  = 16 * 1024
)

// SetTCPBuffSize set TCP connection R/W buff size.
func SetTCPBuffSize(r, w int) error {
	if r < 1 {
		return fmt.Errorf("invalid tcp read buff size: %d", r)
	}
	if w < 1 {
		return fmt.Errorf("invalid tcp write buff size: %d", w)
	}
	DefaultTCPReadBuffSize = r
	DefaultTCPWriteBuffSize = w
	return nil
}
