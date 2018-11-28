package protocol

import (
	"context"
	"log"
	"net"
)

type TcpServerTransport struct {
	addr string
}

func (p *TcpServerTransport) Listen() error {
	listener, err := net.Listen("tcp", p.addr)
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			c, err := listener.Accept()
			if err != nil {
				log.Println("tcp listener break:", err)
				break
			}
			rc := newTcpRConnection(c)
			go func(ctx context.Context, rc *tcpRConnection) {
				if err := rc.rcvLoop(ctx); err != nil {
					log.Println("tcp rconnection error:", err)
				}
			}(ctx, rc)
			go func(ctx context.Context, rc *tcpRConnection) {
				if err := rc.sndLoop(ctx); err != nil {
					log.Println("tcp rconnection error:", err)
				}
			}(ctx, rc)
		}
	}
}

func NewTcpServerTransport(addr string) *TcpServerTransport {
	return &TcpServerTransport{
		addr: addr,
	}
}
