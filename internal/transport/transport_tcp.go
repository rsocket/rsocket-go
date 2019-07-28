package transport

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/pkg/errors"
)

type tcpServerTransport struct {
	network, addr string
	acceptor      ServerTransportAcceptor
	listener      net.Listener
	onceClose     sync.Once
	tls           *tls.Config
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

func (p *tcpServerTransport) Listen(ctx context.Context) (err error) {
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
	return p.listen(ctx)
}

func (p *tcpServerTransport) listen(ctx context.Context) (err error) {
	// Remove unix socket file before exit.
	if p.network == schemaUNIX {
		// Monitor signal of current process and unlink unix socket file.
		go func(sock string) {
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt, syscall.SIGTERM)
			<-c
			_ = p.Close()
		}(p.addr)
	}

	stop := make(chan struct{})
	ctx, cancel := context.WithCancel(ctx)
	go func(ctx context.Context, stop chan struct{}) {
		defer func() {
			_ = p.Close()
			close(stop)
		}()
		<-ctx.Done()
	}(ctx, stop)

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
		go func(ctx context.Context, rawConn net.Conn) {
			conn := newTCPRConnection(rawConn)
			tp := newTransportClient(conn)
			p.acceptor(ctx, tp)
		}(ctx, c)
	}
	cancel()
	<-stop
	return
}

func newTCPServerTransport(network, addr string, c *tls.Config) *tcpServerTransport {
	return &tcpServerTransport{
		network: network,
		addr:    addr,
		tls:     c,
	}
}

func newTCPClientTransport(network, addr string, tlsConfig *tls.Config) (tp *Transport, err error) {
	var rawConn net.Conn
	if tlsConfig == nil {
		rawConn, err = net.Dial(network, addr)
	} else {
		rawConn, err = tls.Dial(network, addr, tlsConfig)
	}
	if err != nil {
		return
	}
	tp = newTransportClient(newTCPRConnection(rawConn))
	return
}
