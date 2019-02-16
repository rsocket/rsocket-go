package rsocket

import (
	"net"
)

func newClientTransportTCP(addr string) (transport, error) {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return newTransportClient(newTcpRConnection(c)), nil
}
