package rsocket

import (
	"context"
	"log"
	"net"
	"sync"
)

type tcpServerTransport struct {
	addr          string
	acceptor      func(setup *FrameSetup, conn RConnection) error
	frameBuffSize int
	listener      net.Listener
	once          *sync.Once
}

func (p *tcpServerTransport) Accept(acceptor func(setup *FrameSetup, conn RConnection) error) {
	p.acceptor = acceptor
}

func (p *tcpServerTransport) Close() (err error) {
	if p.listener == nil {
		return
	}
	p.once.Do(func() {
		err = p.listener.Close()
	})
	return
}

func (p *tcpServerTransport) Listen() (err error) {
	p.listener, err = net.Listen("tcp", p.addr)
	if err != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		if err == nil {
			err = p.Close()
		}
	}()

	for {
		c, err := p.listener.Accept()
		if err != nil {
			log.Println("tcp listener break:", err)
			return err
		}
		rc := newTcpRConnection(c, p.frameBuffSize)
		rc.HandleSetup(func(setup *FrameSetup) (err error) {
			if p.acceptor != nil {
				err = p.acceptor(setup, rc)
			}
			return
		})
		rc.PostFlight(ctx)
	}
}

func newTCPServerTransport(addr string, frameBuffSize int) *tcpServerTransport {
	var size = 64
	if frameBuffSize > 0 {
		size = frameBuffSize
	}
	return &tcpServerTransport{
		addr:          addr,
		frameBuffSize: size,
		once:          &sync.Once{},
	}
}
