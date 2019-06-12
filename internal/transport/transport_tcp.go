package transport

import (
	"context"
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
	onceClose     *sync.Once
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
	l, err := net.Listen(p.network, p.addr)
	if err != nil {
		err = errors.Wrap(err, "server listen failed")
		return
	}
	p.listener = l

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

func newTCPServerTransport(network, addr string) *tcpServerTransport {
	return &tcpServerTransport{
		network:   network,
		addr:      addr,
		onceClose: &sync.Once{},
	}
}

func newTCPClientTransport(network, addr string) (tp *Transport, err error) {
	rawConn, err := net.Dial(network, addr)
	if err != nil {
		return
	}
	conn := newTCPRConnection(rawConn)
	tp = newTransportClient(conn)
	return
}
