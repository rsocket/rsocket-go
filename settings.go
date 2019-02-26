package rsocket

import (
	"fmt"
	"time"
)

const (
	defaultKeepaliveInteval     = 20 * time.Second
	defaultKeepaliveMaxLifetime = 90 * time.Second
)

var (
	defaultConnTcpWriteBuffSize = 16 * 1024
	defaultConnTcpReadBuffSize  = 16 * 1024
	defaultConnSendChanSize     = 64
)

func SetTcpBuffSize(r, w int) error {
	if r < 1 {
		return fmt.Errorf("invalid tcp read buff size: %d", r)
	}
	if w < 1 {
		return fmt.Errorf("invalid tcp write buff size: %d", w)
	}
	defaultConnTcpReadBuffSize = r
	defaultConnTcpWriteBuffSize = w
	return nil
}

func SetSendChanSize(size int) error {
	if size < 1 {
		return fmt.Errorf("invalid send chan size: %d", size)
	}
	defaultConnSendChanSize = size
	return nil
}
