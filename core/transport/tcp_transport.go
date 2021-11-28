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
	mu       sync.Mutex
	m        map[*Transport]struct{}
	f        ListenerFactory
	l        net.Listener
	acceptor ServerTransportAcceptor
	done     chan struct{}
}

func (t *tcpServerTransport) Accept(acceptor ServerTransportAcceptor) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.acceptor = acceptor
}

func (t *tcpServerTransport) Close() (err error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	select {
	case <-t.done:
		// already closed
		break
	default:
		close(t.done)
		if t.l == nil {
			break
		}
		err = t.l.Close()
		for k := range t.m {
			_ = k.Close()
		}
		t.m = nil
	}
	return
}

func (t *tcpServerTransport) Listen(ctx context.Context, notifier chan<- bool) (err error) {
	t.l, err = t.f(ctx)
	if err != nil {
		notifier <- false
		err = errors.Wrap(err, "listen tcp server failed")
		return
	}

	defer func() {
		_ = t.Close()
	}()

	notifier <- true

	// daemon: close if ctx is done.
	go func() {
		select {
		case <-ctx.Done():
			_ = t.Close()
			break
		case <-t.done:
			break
		}
	}()

	// Start loop of accepting connections.
	var c net.Conn
	for {
		c, err = t.l.Accept()
		if err == io.EOF || isClosedErr(err) {
			err = nil
			break
		}
		if err != nil {
			err = errors.Wrap(err, "accept next conn failed")
			break
		}
		// Dispatch raw conn.
		tp := NewTransport(NewTCPConn(c))

		if t.putTransport(tp) {
			go t.acceptor(ctx, tp, func(tp *Transport) {
				t.removeTransport(tp)
			})
		} else {
			_ = t.Close()
		}
	}
	return
}

func (t *tcpServerTransport) removeTransport(tp *Transport) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.m, tp)
}

func (t *tcpServerTransport) putTransport(tp *Transport) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	select {
	case <-t.done:
		// already closed
		return false
	default:
		if t.m == nil {
			return false
		}
		t.m[tp] = struct{}{}
		return true
	}
}

// NewTCPServerTransport creates a new server-side transport.
func NewTCPServerTransport(f ListenerFactory) ServerTransport {
	return &tcpServerTransport{
		f:    f,
		m:    make(map[*Transport]struct{}),
		done: make(chan struct{}),
	}
}

// NewTCPServerTransportWithAddr creates a new server-side transport.
func NewTCPServerTransportWithAddr(network, addr string, tlsConfig *tls.Config) ServerTransport {
	f := func(ctx context.Context) (net.Listener, error) {
		var c net.ListenConfig
		l, err := c.Listen(ctx, network, addr)
		if err != nil {
			return nil, err
		}
		if tlsConfig == nil {
			return l, nil
		}
		return tls.NewListener(l, tlsConfig), nil
	}
	return NewTCPServerTransport(f)
}

// NewTCPClientTransport creates new transport.
func NewTCPClientTransport(c net.Conn) *Transport {
	return NewTransport(NewTCPConn(c))
}

// NewTCPClientTransportWithAddr creates new transport.
func NewTCPClientTransportWithAddr(ctx context.Context, network, addr string, tlsConfig *tls.Config) (tp *Transport, err error) {
	var conn net.Conn
	var dial net.Dialer
	conn, err = dial.DialContext(ctx, network, addr)
	if err != nil {
		return
	}
	if tlsConfig != nil {
		conn = tls.Client(conn, tlsConfig)
	}
	tp = NewTCPClientTransport(conn)
	return
}
