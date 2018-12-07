package rsocket

import (
	"fmt"
	"log"
	"net"
)

type TCPClientTransport struct {
	frameBuffSize int
	addr          string
}

func (p *TCPClientTransport) Connect() (conn RConnection, err error) {
	c, err := net.Dial("tcp", p.addr)
	if err != nil {
		log.Println("dial failed:", err)
		return nil, err
	}
	return newTcpRConnection(c, p.frameBuffSize), nil
}

func newTCPClientTransport(host string, port int) *TCPClientTransport {
	return &TCPClientTransport{
		frameBuffSize: 64,
		addr:          fmt.Sprintf("%s:%d", host, port),
	}
}
