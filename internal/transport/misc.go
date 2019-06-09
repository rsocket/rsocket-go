package transport

import (
	"strings"
)

const (
	tcpWriteBuffSize = 16 * 1024
	tcpReadBuffSize  = 16 * 1024
)

func IsClosedErr(err error) bool {
	if err == nil {
		return false
	}
	if strings.Contains(err.Error(), "use of closed network connection") {
		return true
	}
	return false
}
