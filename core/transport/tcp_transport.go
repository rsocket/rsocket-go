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
	listenerFn func() (net.Listener, error)
	acceptor   ServerTransportAcceptor
	listener   net.Listener
	onceClose  sync.Once
	transports *sync.Map
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
	p.listener, err = p.listenerFn()
	if err != nil {
		close(notifier)
		return
	}
	notifier <- struct{}{}
	close(notifier)
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
		tp := NewTransport(NewTcpConn(c))
		p.transports.Store(tp, struct{}{})
		go p.acceptor(ctx, tp, func(t *Transport) {
			p.transports.Delete(t)
		})
	}
	return
}

func NewTcpServerTransport(gen func() (net.Listener, error)) ServerTransport {
	return &tcpServerTransport{
		listenerFn: gen,
		transports: &sync.Map{},
	}
}

func NewTcpServerTransportWithAddr(network, addr string, c *tls.Config) ServerTransport {
	gen := func() (net.Listener, error) {
		if c == nil {
			return net.Listen(network, addr)
		} else {
			return tls.Listen(network, addr, c)
		}
	}
	return NewTcpServerTransport(gen)
}

func NewTcpClientTransport(rawConn net.Conn) *Transport {
	return NewTransport(NewTcpConn(rawConn))
}

func NewTcpClientTransportWithAddr(network, addr string, tlsConfig *tls.Config) (tp *Transport, err error) {
	var rawConn net.Conn
	if tlsConfig == nil {
		rawConn, err = net.Dial(network, addr)
	} else {
		rawConn, err = tls.Dial(network, addr, tlsConfig)
	}
	if err != nil {
		return
	}
	tp = NewTcpClientTransport(rawConn)
	return
}
