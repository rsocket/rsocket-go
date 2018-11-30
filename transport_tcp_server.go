package rsocket

import (
	"context"
	"log"
	"net"
)

type TcpServerTransport struct {
	addr          string
	acceptor      func(setup *FrameSetup, conn RConnection) error
	frameBuffSize int
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
			rc := newTcpRConnection(c, p.frameBuffSize)
			rc.HandleSetup(func(setup *FrameSetup) (err error) {
				if p.acceptor != nil {
					err = p.acceptor(setup, rc)
				}
				return
			})
			go func(ctx context.Context, rc *tcpRConnection) {
				if err := rc.loopRcv(ctx); err != nil {
					log.Println("tcp rconnection error:", err)
				}
			}(ctx, rc)
			go func(ctx context.Context, rc *tcpRConnection) {
				if err := rc.loopSnd(ctx); err != nil {
					log.Println("tcp rconnection error:", err)
				}
			}(ctx, rc)
		}
	}
}

func newTCPServerTransport(addr string, frameBuffSize int) *TcpServerTransport {
	var size = 64
	if frameBuffSize > 0 {
		size = frameBuffSize
	}
	return &TcpServerTransport{
		addr:          addr,
		frameBuffSize: size,
	}
}
