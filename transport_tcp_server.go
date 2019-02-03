package rsocket

import (
	"context"
	"log"
	"net"
	"sync"
)

type tcpServerTransport struct {
	addr      string
	acceptor  func(setup *frameSetup, conn RConnection) error
	listener  net.Listener
	onceClose *sync.Once
}

func (p *tcpServerTransport) Accept(acceptor func(setup *frameSetup, conn RConnection) error) {
	p.acceptor = acceptor
}

func (p *tcpServerTransport) Close() (err error) {
	if p.listener == nil {
		return
	}
	p.onceClose.Do(func() {
		err = p.listener.Close()
	})
	return
}

func (p *tcpServerTransport) Listen(onReady ...func()) (err error) {
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

	go func() {
		for _, v := range onReady {
			v()
		}
	}()

	for {
		c, err := p.listener.Accept()
		if err != nil {
			log.Println("tcp listener break:", err)
			return err
		}
		rc := newTcpRConnection(c, 0)
		rc.HandleSetup(func(setup *frameSetup) (err error) {
			defer setup.Release()
			if p.acceptor != nil {
				err = p.acceptor(setup, rc)
			}
			return
		})
		rc.PostFlight(ctx)
	}
}

func newTCPServerTransport(addr string) *tcpServerTransport {
	return &tcpServerTransport{
		addr:      addr,
		onceClose: &sync.Once{},
	}
}
