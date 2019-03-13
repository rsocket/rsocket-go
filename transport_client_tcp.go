package rsocket

import (
	"net"
	"time"
)

func newClientTransportTCP(addr string, keepaliveInterval, keepaliveMaxLifetime time.Duration) (transport, error) {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return newTransportClient(newTCPRConnection(c, keepaliveInterval, keepaliveMaxLifetime, true)), nil
}
