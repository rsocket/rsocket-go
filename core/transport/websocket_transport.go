package transport

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/logger"
)

const defaultWebsocketPath = "/"

type wsServerTransport struct {
	upgrader *websocket.Upgrader
	mu       sync.Mutex
	path     string
	acceptor ServerTransportAcceptor
	f        ListenerFactory
	l        net.Listener
	m        map[*Transport]struct{}
	done     chan struct{}
}

func (ws *wsServerTransport) Close() (err error) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	select {
	case <-ws.done:
		// already closed
		break
	default:
		close(ws.done)
		if ws.l == nil {
			break
		}
		// close listener
		err = ws.l.Close()
		// close transports
		for k := range ws.m {
			_ = k.Close()
		}
	}
	return
}

func (ws *wsServerTransport) Accept(acceptor ServerTransportAcceptor) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	ws.acceptor = acceptor
}

func (ws *wsServerTransport) Listen(ctx context.Context, notifier chan<- bool) (err error) {
	ws.l, err = ws.f(ctx)
	if err != nil {
		notifier <- false
		return
	}
	defer func() {
		_ = ws.Close()
	}()

	notifier <- true

	go func() {
		select {
		case <-ctx.Done():
			// context end
			_ = ws.Close()
			break
		case <-ws.done:
			// already closed
			break
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc(ws.path, func(w http.ResponseWriter, r *http.Request) {
		// upgrade websocket
		c, err := ws.upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Errorf("create websocket conn failed: %s\n", err.Error())
			return
		}

		// new websocket transport
		tp := NewTransport(NewWebsocketConnection(c))

		if ws.putTransport(tp) {
			// accept async
			go ws.acceptor(ctx, tp, func(tp *Transport) {
				// remove transport
				ws.removeTransport(tp)
			})
		} else {
			_ = tp.Close()
		}
	})

	err = http.Serve(ws.l, mux)
	if err == io.EOF || isClosedErr(err) {
		err = nil
	} else {
		err = errors.Wrap(err, "listen websocket server failed")
	}
	return
}

func (ws *wsServerTransport) removeTransport(tp *Transport) {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	if ws.m == nil {
		return
	}
	delete(ws.m, tp)
}

func (ws *wsServerTransport) putTransport(tp *Transport) bool {
	ws.mu.Lock()
	defer ws.mu.Unlock()
	select {
	case <-ws.done:
		// already closed
		return false
	default:
		if ws.m == nil {
			return false
		}
		// put transport
		ws.m[tp] = struct{}{}
		return true
	}
}

// NewWebsocketServerTransport creates a new server-side transport.
func NewWebsocketServerTransport(f ListenerFactory, path string, upgrader *websocket.Upgrader) ServerTransport {
	if path == "" {
		path = defaultWebsocketPath
	}
	if upgrader == nil {
		upgrader = &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}
	}
	return &wsServerTransport{
		upgrader: upgrader,
		path:     path,
		f:        f,
		m:        make(map[*Transport]struct{}),
		done:     make(chan struct{}),
	}
}

// NewWebsocketServerTransportWithAddr creates a new server-side transport.
func NewWebsocketServerTransportWithAddr(addr string, path string, upgrader *websocket.Upgrader, config *tls.Config) ServerTransport {
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
	return NewWebsocketServerTransport(f, path, upgrader)
}

// NewWebsocketClientTransport creates a new client-side transport.
func NewWebsocketClientTransport(ctx context.Context, url string, config *tls.Config, header http.Header, proxy func(*http.Request) (*url.URL, error)) (*Transport, error) {
	dial := &websocket.Dialer{
		Proxy:            proxy,
		HandshakeTimeout: 45 * time.Second,
		TLSClientConfig:  config,
	}
	conn, resp, err := dial.DialContext(ctx, url, header)
	if err != nil {
		if resp != nil {
			return nil, errors.Wrapf(err, "dial websocket failed: %s", resp.Status)
		} else {
			return nil, errors.Wrap(err, "dial websocket failed")
		}
	}
	return NewTransport(NewWebsocketConnection(conn)), nil
}
