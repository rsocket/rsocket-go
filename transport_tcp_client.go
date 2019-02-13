package rsocket

import (
	"fmt"
	"log"
	"net"
	"time"
)

type TCPClientTransport struct {
	addr string
}

func (p *TCPClientTransport) Connect(keepaliveInterval time.Duration) (conn RConnection, err error) {
	c, err := net.Dial("tcp", p.addr)
	if err != nil {
		log.Println("dial failed:", err)
		return nil, err
	}
	return newTcpRConnection(c, keepaliveInterval), nil
}

func newTCPClientTransport(host string, port int) *TCPClientTransport {
	return &TCPClientTransport{
		addr: fmt.Sprintf("%s:%d", host, port),
	}
}
