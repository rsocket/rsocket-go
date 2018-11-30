package rsocket

import "net"

type TCPClientTransport struct {
	frameBuffSize int
}

func (p *TCPClientTransport) Connect(addr string) (conn *tcpRConnection, err error) {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return nil, err
	}
	return newTcpRConnection(c, p.frameBuffSize), nil
}

func newTCPClientTransport() *TCPClientTransport {
	return &TCPClientTransport{
		frameBuffSize: 64,
	}
}
