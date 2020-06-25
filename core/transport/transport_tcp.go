package transport

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"sync"

	"github.com/pkg/errors"
)

type tcpServerTransport struct {
	network, addr string
	acceptor      ServerTransportAcceptor
	listener      net.Listener
	onceClose     sync.Once
	tls           *tls.Config
	transports    *sync.Map
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

		p.transports.Range(func(key, value interface{}) bool {
			_ = key.(*Transport).Close()
			return true
		})

	})
	return
}

func (p *tcpServerTransport) Listen(ctx context.Context, notifier chan<- struct{}) (err error) {
	if p.tls == nil {
		p.listener, err = net.Listen(p.network, p.addr)
		if err != nil {
			err = errors.Wrap(err, "server listen failed")
			return
		}
	} else {
		p.listener, err = tls.Listen(p.network, p.addr, p.tls)
		if err != nil {
			err = errors.Wrap(err, "server listen failed")
			return
		}
	}
	notifier <- struct{}{}
	return p.listen(ctx)
}

func (p *tcpServerTransport) listen(ctx context.Context) (err error) {
	done := make(chan struct{})

	defer func() {
		close(done)
		_ = p.Close()
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				_ = p.Close()
				return
			case <-done:
				return
			}
		}
	}()

	// Start loop of accepting connections.
	var c net.Conn
	for {
		c, err = p.listener.Accept()
		if err == io.EOF || isClosedErr(err) {
			err = nil
			break
		}
		if err != nil {
			err = errors.Wrap(err, "accept next conn failed")
			break
		}
		// Dispatch raw conn.
		tp := NewTransport(newTCPRConnection(c))
		p.transports.Store(tp, struct{}{})
		go p.acceptor(ctx, tp, func(t *Transport) {
			p.transports.Delete(t)
		})
	}
	return
}

func NewTcpServerTransport(network, addr string, c *tls.Config) *tcpServerTransport {
	return &tcpServerTransport{
		network:    network,
		addr:       addr,
		tls:        c,
		transports: &sync.Map{},
	}
}

func NewTcpClientTransport(network, addr string, tlsConfig *tls.Config) (tp *Transport, err error) {
	var rawConn net.Conn
	if tlsConfig == nil {
		rawConn, err = net.Dial(network, addr)
	} else {
		rawConn, err = tls.Dial(network, addr, tlsConfig)
	}
	if err != nil {
		return
	}
	tp = NewTransport(newTCPRConnection(rawConn))
	return
}
