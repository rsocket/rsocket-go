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
	path       string
	acceptor   ServerTransportAcceptor
	onceClose  sync.Once
	listenerFn func() (net.Listener, error)
	listener   net.Listener
	transports *sync.Map
}

func (p *wsServerTransport) Close() (err error) {
	p.onceClose.Do(func() {
		if p.listener != nil {
			err = p.listener.Close()
		}
	})
	return
}

func (p *wsServerTransport) Accept(acceptor ServerTransportAcceptor) {
	p.acceptor = acceptor
}

func (p *wsServerTransport) Listen(ctx context.Context, notifier chan<- struct{}) (err error) {
	mux := http.NewServeMux()
	mux.HandleFunc(p.path, func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Errorf("create websocket conn failed: %s\n", err.Error())
			return
		}
		tp := NewTransport(NewWebsocketConnection(c))
		p.transports.Store(tp, struct{}{})
		go p.acceptor(ctx, tp, func(tp *Transport) {
			p.transports.Delete(tp)
		})
	})

	p.listener, err = p.listenerFn()

	if err != nil {
		err = errors.Wrap(err, "server listen failed")
		close(notifier)
		return
	}

	notifier <- struct{}{}

	stop := make(chan struct{})
	ctx, cancel := context.WithCancel(ctx)

	go func(ctx context.Context, stop chan struct{}) {
		defer func() {
			_ = p.Close()
			close(stop)
		}()
		<-ctx.Done()
	}(ctx, stop)

	err = http.Serve(p.listener, mux)
	if err == io.EOF || isClosedErr(err) {
		err = nil
	} else {
		err = errors.Wrap(err, "listen websocket server failed")
	}
	cancel()
	<-stop
	return
}

func NewWebsocketServerTransport(gen func() (net.Listener, error), path string) ServerTransport {
	if path == "" {
		path = defaultWebsocketPath
	}
	return &wsServerTransport{
		path:       path,
		listenerFn: gen,
		transports: &sync.Map{},
	}
}

func NewWebsocketServerTransportWithAddr(addr string, path string, c *tls.Config) ServerTransport {
	return NewWebsocketServerTransport(func() (net.Listener, error) {
		if c == nil {
			return net.Listen("tcp", addr)
		}
		return tls.Listen("tcp", addr, c)
	}, path)
}

func NewWebsocketClientTransport(url string, tc *tls.Config, header http.Header) (*Transport, error) {
	var d *websocket.Dialer
	if tc == nil {
		d = websocket.DefaultDialer
	} else {
		d = &websocket.Dialer{
			Proxy:            http.ProxyFromEnvironment,
			HandshakeTimeout: 45 * time.Second,
			TLSClientConfig:  tc,
		}
	}
	wsConn, _, err := d.Dial(url, header)
	if err != nil {
		return nil, errors.Wrap(err, "dial websocket failed")
	}
	return NewTransport(NewWebsocketConnection(wsConn)), nil
}
