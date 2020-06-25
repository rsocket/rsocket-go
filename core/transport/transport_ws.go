package transport

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/logger"
)

const defaultWebsocketPath = "/"

var upgrader websocket.Upgrader

func init() {
	// Default allow CORS.
	cors := true
	if v, ok := os.LookupEnv("RSOCKET_WS_CORS"); ok {
		v = strings.TrimSpace(strings.ToLower(v))
		cors = v == "yes" || v == "on" || v == "1" || v == "true"
	}
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}
	if cors {
		upgrader.CheckOrigin = func(r *http.Request) bool {
			return true
		}
	}
}

type wsServerTransport struct {
	addr       string
	path       string
	acceptor   ServerTransportAcceptor
	onceClose  sync.Once
	listener   net.Listener
	tls        *tls.Config
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

		tp := NewTransport(newWebsocketConnection(c))
		p.transports.Store(tp, struct{}{})
		go p.acceptor(ctx, tp, func(tp *Transport) {
			p.transports.Delete(tp)
		})
	})

	if p.tls == nil {
		p.listener, err = net.Listen("tcp", p.addr)
	} else {
		p.listener, err = tls.Listen("tcp", p.addr, p.tls)
	}

	if err != nil {
		err = errors.Wrap(err, "server listen failed")
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

func NewWebsocketServerTransport(addr string, path string, c *tls.Config) *wsServerTransport {
	if path == "" {
		path = defaultWebsocketPath
	}
	return &wsServerTransport{
		addr:       addr,
		path:       path,
		tls:        c,
		transports: &sync.Map{},
	}
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
	return NewTransport(newWebsocketConnection(wsConn)), nil
}
