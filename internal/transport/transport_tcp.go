package transport

import (
	"context"
	"net"
	"sync"

	"github.com/rsocket/rsocket-go/internal/logger"
)

type tcpServerTransport struct {
	addr      string
	acceptor  ServerTransportAcceptor
	listener  net.Listener
	onceClose *sync.Once
}

func (p *tcpServerTransport) Accept(acceptor ServerTransportAcceptor) {
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
			logger.Errorf("protoTCP listener break: %s\n", err)
			return err
		}
		go func(ctx context.Context, rawConn net.Conn) {
			conn := newTCPRConnection(rawConn)
			tp := newTransportClient(conn)
			p.acceptor(ctx, tp)
		}(ctx, c)
	}
}

func newTCPServerTransport(addr string) *tcpServerTransport {
	return &tcpServerTransport{
		addr:      addr,
		onceClose: &sync.Once{},
	}
}

func newTCPClientTransport(addr string) (tp *Transport, err error) {
	rawConn, err := net.Dial("tcp", addr)
	if err != nil {
		return
	}
	conn := newTCPRConnection(rawConn)
	tp = newTransportClient(conn)
	return
}
