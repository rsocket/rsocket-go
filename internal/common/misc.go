package common

import (
	"time"
	"unsafe"
)

const (
	// DefaultKeepaliveInteval is default keepalive interval duration.
	DefaultKeepaliveInteval = 20 * time.Second
	// DefaultKeepaliveMaxLifetime is default keepalive max lifetime.
	DefaultKeepaliveMaxLifetime = 90 * time.Second
)

// Str2bytes convert string to bytes.
func Str2bytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}

// Bytes2str convert bytes to string.
func Bytes2str(b []byte) (s string) {
	if len(b) < 1 {
		return
	}
	s = *(*string)(unsafe.Pointer(&b))
	return
}
