package transport

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/logger"
)

const defaultWebsocketPath = "/"

var upgrader websocket.Upgrader

func init() {
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
}

type wsServerTransport struct {
	mu       sync.Mutex
	path     string
	acceptor ServerTransportAcceptor
	f        ListenerFactory
	l        net.Listener
	m        map[*Transport]struct{}
	done     chan struct{}
}

func (p *wsServerTransport) Close() (err error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	select {
	case <-p.done:
		// already closed
		break
	default:
		close(p.done)
		if p.l == nil {
			break
		}
		// close listener
		err = p.l.Close()
		// close transports
		for k := range p.m {
			_ = k.Close()
		}
	}
	return
}

func (p *wsServerTransport) Accept(acceptor ServerTransportAcceptor) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.acceptor = acceptor
}

func (p *wsServerTransport) Listen(ctx context.Context, notifier chan<- bool) (err error) {
	p.l, err = p.f(ctx)
	if err != nil {
		notifier <- false
		return
	}
	defer func() {
		_ = p.Close()
	}()

	notifier <- true

	mux := http.NewServeMux()
	mux.HandleFunc(p.path, func(w http.ResponseWriter, r *http.Request) {
		// upgrade websocket
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Errorf("create websocket conn failed: %s\n", err.Error())
			return
		}

		p.mu.Lock()
		defer p.mu.Unlock()

		select {
		case <-p.done:
			// already closed
			return
		default:
			if p.m == nil {
				return
			}
		}

		// new websocket transport
		tp := NewTransport(NewWebsocketConnection(c))

		// put transport
		p.m[tp] = struct{}{}

		// accept async
		go p.acceptor(ctx, tp, func(tp *Transport) {
			// remove transport
			p.mu.Lock()
			defer p.mu.Unlock()
			delete(p.m, tp)
		})
	})

	err = http.Serve(p.l, mux)
	if err == io.EOF || isClosedErr(err) {
		err = nil
	} else {
		err = errors.Wrap(err, "listen websocket server failed")
	}
	return
}

func NewWebsocketServerTransport(f ListenerFactory, path string) ServerTransport {
	if path == "" {
		path = defaultWebsocketPath
	}
	return &wsServerTransport{
		path: path,
		f:    f,
		m:    make(map[*Transport]struct{}),
		done: make(chan struct{}),
	}
}

func NewWebsocketServerTransportWithAddr(addr string, path string, config *tls.Config) ServerTransport {
	f := func(ctx context.Context) (net.Listener, error) {
		var c net.ListenConfig
		l, err := c.Listen(ctx, "tcp", addr)
		if err != nil {
			return nil, err
		}
		if config == nil {
			return l, nil
		}
		return tls.NewListener(l, config), nil
	}
	return NewWebsocketServerTransport(f, path)
}

func NewWebsocketClientTransport(url string, config *tls.Config, header http.Header) (*Transport, error) {
	var d *websocket.Dialer
	if config == nil {
		d = websocket.DefaultDialer
	} else {
		d = &websocket.Dialer{
			Proxy:            http.ProxyFromEnvironment,
			HandshakeTimeout: 45 * time.Second,
			TLSClientConfig:  config,
		}
	}
	wsConn, _, err := d.Dial(url, header)
	if err != nil {
		return nil, errors.Wrap(err, "dial websocket failed")
	}
	return NewTransport(NewWebsocketConnection(wsConn)), nil
}
