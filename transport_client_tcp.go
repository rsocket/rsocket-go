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
	return newTransportClient(newTcpRConnection(c, keepaliveInterval, keepaliveMaxLifetime, true)), nil
}
