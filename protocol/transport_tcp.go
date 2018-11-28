package protocol

import (
	"context"
	"log"
	"net"
)

type TcpServerTransport struct {
	addr     string
	acceptor func(setup *FrameSetup, conn RConnection) error
}

func (p *TcpServerTransport) Accept(acceptor func(setup *FrameSetup, conn RConnection) error) {
	p.acceptor = acceptor
}

func (p *TcpServerTransport) Listen(ctx context.Context) error {
	listener, err := net.Listen("tcp", p.addr)
	if err != nil {
		return err
	}
	defer func() {
		if err := listener.Close(); err != nil {
			log.Println("close listener failed:", err)
		}
	}()

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
			rc.HandleSetup(func(setup *FrameSetup) (err error) {
				if p.acceptor != nil {
					err = p.acceptor(setup, rc)
				}
				return
			})
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
