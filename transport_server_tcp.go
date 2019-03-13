package rsocket

import (
	"context"
	"net"
	"sync"
)

type tcpServerTransport struct {
	addr      string
	acceptor  func(setup *frameSetup, tp transport) error
	listener  net.Listener
	onceClose *sync.Once
}

func (p *tcpServerTransport) Accept(acceptor func(setup *frameSetup, tp transport) error) {
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
			logger.Errorf("TCP listener break: %s\n", err)
			return err
		}
		go func(ctx context.Context, c net.Conn) {
			select {
			case <-ctx.Done():
				_ = c.Close()
				return
			default:
				tp := newTransportClient(newTCPRConnection(c, defaultKeepaliveInteval, defaultKeepaliveMaxLifetime, false))
				tp.handleSetup(func(f Frame) (err error) {
					setup := f.(*frameSetup)
					defer setup.Release()
					if p.acceptor != nil {
						err = p.acceptor(setup, tp)
					}
					return
				})
				_ = tp.Start(ctx)
			}
		}(ctx, c)
	}
}

func newTCPServerTransport(addr string) *tcpServerTransport {
	return &tcpServerTransport{
		addr:      addr,
		onceClose: &sync.Once{},
	}
}
