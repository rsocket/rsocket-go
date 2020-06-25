package rsocket

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/core/transport"
)

type Transporter interface {
	Client() transport.ToClientTransport
	Server() transport.ToServerTransport
}

type tcpTransporter struct {
	addr string
	tls  *tls.Config
}

type TcpTransporterBuilder struct {
	opts []func(*tcpTransporter)
}

func (t *tcpTransporter) Server() transport.ToServerTransport {
	return func(ctx context.Context) (transport.ServerTransport, error) {
		return transport.NewTcpServerTransport("tcp", t.addr, t.tls), nil
	}
}

func (t *tcpTransporter) Client() transport.ToClientTransport {
	return func(ctx context.Context) (*transport.Transport, error) {
		return transport.NewTcpClientTransport("tcp", t.addr, t.tls)
	}
}

func (t *TcpTransporterBuilder) Addr(addr string) *TcpTransporterBuilder {
	t.opts = append(t.opts, func(transporter *tcpTransporter) {
		transporter.addr = addr
	})
	return t
}

func (t *TcpTransporterBuilder) HostAndPort(host string, port int) *TcpTransporterBuilder {
	return t.Addr(fmt.Sprintf("%s:%d", host, port))
}

func (t *TcpTransporterBuilder) TLS(config *tls.Config) *TcpTransporterBuilder {
	t.opts = append(t.opts, func(transporter *tcpTransporter) {
		transporter.tls = config
	})
	return t
}

func (t *TcpTransporterBuilder) Build() Transporter {
	tp := &tcpTransporter{
		addr: ":7878",
		tls:  nil,
	}
	for _, opt := range t.opts {
		opt(tp)
	}
	return tp
}

type wsTransporter struct {
	url    string
	tls    *tls.Config
	header http.Header
}

type WebsocketTransporterBuilder struct {
	opts []func(*wsTransporter)
}

func (w *WebsocketTransporterBuilder) Header(header http.Header) *WebsocketTransporterBuilder {
	w.opts = append(w.opts, func(transporter *wsTransporter) {
		transporter.header = header
	})
	return w
}

func (w *WebsocketTransporterBuilder) Url(url string) *WebsocketTransporterBuilder {
	w.opts = append(w.opts, func(transporter *wsTransporter) {
		transporter.url = url
	})
	return w
}

func (w *WebsocketTransporterBuilder) TLS(config *tls.Config) *WebsocketTransporterBuilder {
	w.opts = append(w.opts, func(transporter *wsTransporter) {
		transporter.tls = config
	})
	return w
}

func (w *WebsocketTransporterBuilder) Build() Transporter {
	ws := &wsTransporter{
		url: "",
	}
	for _, opt := range w.opts {
		opt(ws)
	}
	return ws
}

func (w *wsTransporter) Server() transport.ToServerTransport {
	return func(ctx context.Context) (transport.ServerTransport, error) {
		u, err := url.Parse(w.url)
		if err != nil {
			return nil, err
		}
		port := u.Port()
		if len(port) < 1 {
			return nil, errors.New("missing websocket port")
		}
		return transport.NewWebsocketServerTransport(fmt.Sprintf("%s:%s", u.Hostname(), port), u.Path, w.tls), nil
	}
}

func (w *wsTransporter) Client() transport.ToClientTransport {
	return func(ctx context.Context) (*transport.Transport, error) {
		return transport.NewWebsocketClientTransport(w.url, w.tls, w.header)
	}
}

type UnixTransporter struct {
	path string
}

type UnixTransporterBuilder struct {
	opts []func(*UnixTransporter)
}

func (u *UnixTransporter) Server() transport.ToServerTransport {
	return func(ctx context.Context) (transport.ServerTransport, error) {
		if _, err := os.Stat(u.path); !os.IsNotExist(err) {
			return nil, err
		}
		return transport.NewTcpServerTransport("unix", u.path, nil), nil
	}
}

func (u *UnixTransporter) Client() transport.ToClientTransport {
	return func(ctx context.Context) (*transport.Transport, error) {
		return transport.NewTcpClientTransport("unix", u.path, nil)
	}
}

func (u *UnixTransporterBuilder) Path(path string) *UnixTransporterBuilder {
	u.opts = append(u.opts, func(transporter *UnixTransporter) {
		transporter.path = path
	})
	return u
}

func (u *UnixTransporterBuilder) Build() Transporter {
	tp := &UnixTransporter{
		path: "/var/run/rsocket.sock",
	}
	for _, opt := range u.opts {
		opt(tp)
	}
	return tp
}

func Tcp() *TcpTransporterBuilder {
	return &TcpTransporterBuilder{}
}

func Websocket() *WebsocketTransporterBuilder {
	return &WebsocketTransporterBuilder{}
}

func Unix() *UnixTransporterBuilder {
	return &UnixTransporterBuilder{}
}
