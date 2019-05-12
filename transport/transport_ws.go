package transport

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rsocket/rsocket-go/common"
	"github.com/rsocket/rsocket-go/common/logger"
	"github.com/rsocket/rsocket-go/framing"
)

const defaultWebsocketPath = "/"

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type wsServerTransport struct {
	addr      string
	path      string
	acceptor  setupAcceptor
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

func (p *wsServerTransport) Accept(acceptor func(setup *framing.FrameSetup, conn Transport) error) {
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

	mux := http.NewServeMux()
	mux.HandleFunc(p.path, func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			logger.Errorf("create websocket conn failed: %s\n", err.Error())
			return
		}
		go func(c *websocket.Conn) {
			conn := newWebsocketConnection(c, common.DefaultKeepaliveInteval, common.DefaultKeepaliveMaxLifetime, false)
			tp := newTransportClient(conn)
			tp.HandleSetup(func(f framing.Frame) (err error) {
				setup := f.(*framing.FrameSetup)
				defer setup.Release()
				if p.acceptor != nil {
					err = p.acceptor(setup, tp)
				}
				return
			})
			if err := tp.Start(context.Background()); err != nil {
				if logger.IsDebugEnabled() {
					logger.Debugf("transport exit: %s\n", err.Error())
				}
			}
		}(c)
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

func newWebsocketClientTransport(url string, keepaliveInterval, keepaliveMaxLifetime time.Duration) (tp Transport, err error) {
	var wsConn *websocket.Conn
	wsConn, _, err = websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return
	}
	c := newWebsocketConnection(wsConn, keepaliveInterval, keepaliveMaxLifetime, true)
	tp = newTransportClient(c)
	return
}
