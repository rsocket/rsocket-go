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
