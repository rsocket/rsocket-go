package transport

import (
	"context"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rsocket/rsocket-go/internal/logger"
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

func (p *wsServerTransport) Listen(onReady ...func()) (err error) {
	defer func() {
		if err == nil {
			err = p.Close()
		}
	}()

	go func() {
		for _, v := range onReady {
			v()
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
	}()

	mux := http.NewServeMux()
	mux.HandleFunc(p.path, func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Errorf("create websocket conn failed: %s\n", err.Error())
			return
		}
		go func(c *websocket.Conn, ctx context.Context) {
			conn := newWebsocketConnection(c)
			conn.SetCounter(NewCounter())
			tp := newTransportClient(conn)
			p.acceptor(ctx, tp)
			if err := tp.Start(ctx); err != nil {
				if logger.IsDebugEnabled() {
					logger.Debugf("transport exit: %s\n", err.Error())
				}
			}
		}(c, ctx)
	})
	p.server = &http.Server{
		Addr:    p.addr,
		Handler: mux,
	}
	err = p.server.ListenAndServe()
	return
}

func newWebsocketServerTransport(addr string, path string) *wsServerTransport {
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
