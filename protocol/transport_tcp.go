package protocol

import (
	"log"
	"net"
)

type TcpServerTransport struct {
	addr string
}

func (p *TcpServerTransport) Listen() {
	listener, err := net.Listen("tcp", p.addr)
	if err != nil {
		panic(err)
	}
	for {
		select {
		default:
			c, err := listener.Accept()
			if err != nil {
				log.Println("tcp listener break:", err)
				break
			}
			go func(rc *tcpRConnection) {
				if err := rc.poll(); err != nil {
					log.Println("tcp rconnection error:", err)
				}
			}(newTcpRConnection(c))
		}
	}
}

func NewTcpServerTransport(addr string) *TcpServerTransport {
	return &TcpServerTransport{
		addr: addr,
	}
}

type tcpRConnection struct {
	c       net.Conn
	decoder *tcpFrameDecoder
}

func (p *tcpRConnection) Close() error {
	panic("implement me")
}

func (p *tcpRConnection) Send(first *Frame, others ...*Frame) error {
	panic("implement me")
}

func (p *tcpRConnection) Receive() (*Frame, error) {
	panic("implement me")
}

func (p *tcpRConnection) poll() error {
	p.decoder.sliceFrameLength()
	return nil
}

func newTcpRConnection(c net.Conn) *tcpRConnection {
	return &tcpRConnection{
		c:       c,
		decoder: newTcpFrameDecoder(c),
	}
}
