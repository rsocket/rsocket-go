package transport

import (
	"net"
	"time"
)

// NewClientTransportTCP returns a new client transport on protoTCP networking.
func NewClientTransportTCP(addr string, keepaliveInterval, keepaliveMaxLifetime time.Duration) (Transport, error) {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return newTransportClient(newTCPRConnection(c, keepaliveInterval, keepaliveMaxLifetime, true)), nil
}
