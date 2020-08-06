package rsocket

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	"github.com/rsocket/rsocket-go/core/transport"
)

const DefaultUnixSockPath = "/var/run/rsocket.sock"
const DefaultPort = 7878

type TcpClientBuilder struct {
	addr   string
	tlsCfg *tls.Config
}

type TcpServerBuilder struct {
	addr   string
	tlsCfg *tls.Config
}

type WebsocketClientBuilder struct {
	url    string
	tlsCfg *tls.Config
	header http.Header
}

type WebsocketServerBuilder struct {
	addr      string
	path      string
	tlsConfig *tls.Config
}

type UnixClientBuilder struct {
	path string
}

type UnixServerBuilder struct {
	path string
}

func (us *UnixServerBuilder) SetPath(path string) *UnixServerBuilder {
	us.path = path
	return us
}

func (us *UnixServerBuilder) Build() transport.ServerTransportFunc {
	return func(ctx context.Context) (transport.ServerTransport, error) {
		if _, err := os.Stat(us.path); !os.IsNotExist(err) {
			return nil, err
		}
		return transport.NewTcpServerTransportWithAddr("unix", us.path, nil), nil
	}
}

func (uc *UnixClientBuilder) SetPath(path string) *UnixClientBuilder {
	uc.path = path
	return uc
}

func (uc UnixClientBuilder) Build() transport.ClientTransportFunc {
	return func(ctx context.Context) (*transport.Transport, error) {
		return transport.NewTcpClientTransportWithAddr("unix", uc.path, nil)
	}
}

func (ws *WebsocketServerBuilder) SetAddr(addr string) *WebsocketServerBuilder {
	ws.addr = addr
	return ws
}

func (ws *WebsocketServerBuilder) SetPath(path string) *WebsocketServerBuilder {
	ws.path = path
	return ws
}

func (ws *WebsocketServerBuilder) SetTlsConfig(c *tls.Config) *WebsocketServerBuilder {
	ws.tlsConfig = c
	return ws
}

func (ws *WebsocketServerBuilder) Build() transport.ServerTransportFunc {
	return func(ctx context.Context) (transport.ServerTransport, error) {
		return transport.NewWebsocketServerTransportWithAddr(ws.addr, ws.path, ws.tlsConfig), nil
	}
}

func (wc *WebsocketClientBuilder) SetTlsConfig(c *tls.Config) *WebsocketClientBuilder {
	wc.tlsCfg = c
	return wc
}

func (wc *WebsocketClientBuilder) SetUrl(url string) *WebsocketClientBuilder {
	wc.url = url
	return wc
}

func (wc *WebsocketClientBuilder) SetHeader(h http.Header) *WebsocketClientBuilder {
	wc.header = h
	return wc
}

func (wc *WebsocketClientBuilder) Build() transport.ClientTransportFunc {
	return func(ctx context.Context) (*transport.Transport, error) {
		return transport.NewWebsocketClientTransport(wc.url, wc.tlsCfg, wc.header)
	}
}

func (ts *TcpServerBuilder) SetHostAndPort(host string, port int) *TcpServerBuilder {
	ts.addr = fmt.Sprintf("%s:%d", host, port)
	return ts
}

func (ts *TcpServerBuilder) SetAddr(addr string) *TcpServerBuilder {
	ts.addr = addr
	return ts
}

func (ts *TcpServerBuilder) SetTlsConfig(c *tls.Config) *TcpServerBuilder {
	ts.tlsCfg = c
	return ts
}

func (ts *TcpServerBuilder) Build() transport.ServerTransportFunc {
	return func(ctx context.Context) (transport.ServerTransport, error) {
		return transport.NewTcpServerTransportWithAddr("tcp", ts.addr, ts.tlsCfg), nil
	}
}

func (tc *TcpClientBuilder) SetHostAndPort(host string, port int) *TcpClientBuilder {
	tc.addr = fmt.Sprintf("%s:%d", host, port)
	return tc
}

func (tc *TcpClientBuilder) SetAddr(addr string) *TcpClientBuilder {
	tc.addr = addr
	return tc
}

func (tc *TcpClientBuilder) SetTlsConfig(c *tls.Config) *TcpClientBuilder {
	tc.tlsCfg = c
	return tc
}

func (tc *TcpClientBuilder) Build() transport.ClientTransportFunc {
	return func(ctx context.Context) (*transport.Transport, error) {
		return transport.NewTcpClientTransportWithAddr("tcp", tc.addr, tc.tlsCfg)
	}
}

func TcpClient() *TcpClientBuilder {
	return &TcpClientBuilder{
		addr: fmt.Sprintf(":%d", DefaultPort),
	}
}

func TcpServer() *TcpServerBuilder {
	return &TcpServerBuilder{
		addr: fmt.Sprintf(":%d", DefaultPort),
	}
}

func WebsocketClient() *WebsocketClientBuilder {
	return &WebsocketClientBuilder{
		url: fmt.Sprintf("ws://127.0.0.1:%d", DefaultPort),
	}
}

func WebsocketServer() *WebsocketServerBuilder {
	return &WebsocketServerBuilder{
		addr: fmt.Sprintf(":%d", DefaultPort),
		path: "/",
	}
}

func UnixClient() *UnixClientBuilder {
	return &UnixClientBuilder{
		path: DefaultUnixSockPath,
	}
}

func UnixServer() *UnixServerBuilder {
	return &UnixServerBuilder{
		path: DefaultUnixSockPath,
	}
}
