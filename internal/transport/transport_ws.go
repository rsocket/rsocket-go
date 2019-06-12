package transport

import (
	"context"
	"io"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/logger"
)

const defaultWebsocketPath = "/"

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type wsServerTransport struct {
	addr      string
	path      string
	acceptor  ServerTransportAcceptor
	onceClose *sync.Once
	server    *http.Server
}

func (p *wsServerTransport) Close() (err error) {
	p.onceClose.Do(func() {
		if p.server != nil {
			err = p.server.Shutdown(context.Background())
		}
	})
	return
}

func (p *wsServerTransport) Accept(acceptor ServerTransportAcceptor) {
	p.acceptor = acceptor
}

func (p *wsServerTransport) Listen(ctx context.Context) (err error) {
	mux := http.NewServeMux()
	mux.HandleFunc(p.path, func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Errorf("create websocket conn failed: %s\n", err.Error())
			return
		}
		go func(c *websocket.Conn, ctx context.Context) {
			conn := newWebsocketConnection(c)
			tp := newTransportClient(conn)
			p.acceptor(ctx, tp)
		}(c, ctx)
	})
	p.server = &http.Server{
		Addr:    p.addr,
		Handler: mux,
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

	err = p.server.ListenAndServe()
	if err == io.EOF || isClosedErr(err) {
		err = nil
	} else {
		err = errors.Wrap(err, "listen websocket server failed")
	}
	cancel()
	<-stop
	return
}

func newWebsocketServerTransport(addr string, path string) *wsServerTransport {
	if path == "" {
		path = defaultWebsocketPath
	}
	return &wsServerTransport{
		addr:      addr,
		path:      path,
		onceClose: &sync.Once{},
	}
}

func newWebsocketClientTransport(url string) (tp *Transport, err error) {
	var wsConn *websocket.Conn
	wsConn, _, err = websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return
	}
	c := newWebsocketConnection(wsConn)
	tp = newTransportClient(c)
	return
}
