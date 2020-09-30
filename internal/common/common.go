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

type Releasable interface {
	Release()
}

func TryRelease(input interface{}) {
	if input == nil {
		return
	}
	if r, ok := input.(Releasable); ok {
		r.Release()
	}
}

func CloneBytes(b []byte) []byte {
	if b == nil || len(b) < 1 {
		return nil
	}
	clone := make([]byte, len(b))
	copy(clone, b)
	return clone
}

func SafeCloseDoneChan(c chan<- struct{}) (ok bool) {
	defer func() {
		ok = recover() == nil
	}()
	close(c)
	return
}
