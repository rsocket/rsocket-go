package rsocket

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"

	"github.com/gorilla/websocket"
	"github.com/rsocket/rsocket-go/core/transport"
)

const (
	// DefaultUnixSockPath is the default UDS sock file path.
	DefaultUnixSockPath = "/var/run/rsocket.sock"
	// DefaultPort is the default port RSocket used.
	DefaultPort = 7878
)

// TCPClientBuilder provides builder which can be used to create a client-side TCP transport easily.
type TCPClientBuilder struct {
	addr   string
	tlsCfg *tls.Config
}

// TCPServerBuilder provides builder which can be used to create a server-side TCP transport easily.
type TCPServerBuilder struct {
	addr   string
	tlsCfg *tls.Config
}

// WebsocketClientBuilder provides builder which can be used to create a client-side Websocket transport easily.
type WebsocketClientBuilder struct {
	url    string
	tlsCfg *tls.Config
	header http.Header
	proxy  func(*http.Request) (*url.URL, error)
}

// WebsocketServerBuilder provides builder which can be used to create a server-side Websocket transport easily.
type WebsocketServerBuilder struct {
	addr      string
	path      string
	tlsConfig *tls.Config
	upgrader  *websocket.Upgrader
}

// UnixClientBuilder provides builder which can be used to create a client-side UDS transport easily.
type UnixClientBuilder struct {
	path string
}

// UnixServerBuilder provides builder which can be used to create a server-side UDS transport easily.
type UnixServerBuilder struct {
	path string
}

// SetPath sets UDS sock file path.
func (us *UnixServerBuilder) SetPath(path string) *UnixServerBuilder {
	us.path = path
	return us
}

// Build builds and returns a new ServerTransporter.
func (us *UnixServerBuilder) Build() transport.ServerTransporter {
	return func(ctx context.Context) (transport.ServerTransport, error) {
		if _, err := os.Stat(us.path); !os.IsNotExist(err) {
			return nil, err
		}
		return transport.NewTCPServerTransportWithAddr("unix", us.path, nil), nil
	}
}

// SetPath sets UDS sock file path.
func (uc *UnixClientBuilder) SetPath(path string) *UnixClientBuilder {
	uc.path = path
	return uc
}

// Build builds and returns a new ClientTransporter.
func (uc UnixClientBuilder) Build() transport.ClientTransporter {
	return func(ctx context.Context) (*transport.Transport, error) {
		return transport.NewTCPClientTransportWithAddr(ctx, "unix", uc.path, nil)
	}
}

// SetAddr sets the websocket listen addr. Default addr is "127.0.0.1:7878".
func (ws *WebsocketServerBuilder) SetAddr(addr string) *WebsocketServerBuilder {
	ws.addr = addr
	return ws
}

// SetPath sets the path of websocket.
func (ws *WebsocketServerBuilder) SetPath(path string) *WebsocketServerBuilder {
	ws.path = path
	return ws
}

// SetTLSConfig sets the tls config.
//
// You can generate cert.pem and key.pem for local testing:
//
//	 go run $GOROOT/src/crypto/tls/generate_cert.go --host localhost
//
//	 Load X509
//	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
//	if err != nil {
//		panic(err)
//	}
//	// Init TLS configuration.
//	tc := &tls.Config{
//		MinVersion:               tls.VersionTLS12,
//		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
//		PreferServerCipherSuites: true,
//		CipherSuites: []uint16{
//			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
//			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
//			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
//			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
//		},
//		Certificates: []tls.Certificate{cert},
//	}
func (ws *WebsocketServerBuilder) SetTLSConfig(c *tls.Config) *WebsocketServerBuilder {
	ws.tlsConfig = c
	return ws
}

// SetUpgrader sets websocket upgrader.
// You can customize your own websocket upgrader instead of the default upgrader.
//
// Example(also the default value):
// upgrader := &websocket.Upgrader{
//		ReadBufferSize:  1024,
//		WriteBufferSize: 1024,
//		CheckOrigin: func(r *http.Request) bool {
//			return true
//		},
// }
func (ws *WebsocketServerBuilder) SetUpgrader(upgrader *websocket.Upgrader) *WebsocketServerBuilder {
	ws.upgrader = upgrader
	return ws
}

// Build builds and returns a new websocket ServerTransporter.
func (ws *WebsocketServerBuilder) Build() transport.ServerTransporter {
	return func(ctx context.Context) (transport.ServerTransport, error) {
		return transport.NewWebsocketServerTransportWithAddr(ws.addr, ws.path, ws.upgrader, ws.tlsConfig), nil
	}
}

// SetTLSConfig sets the tls config.
//
// Here's an example:
//
// tc := &tls.Config{
//	InsecureSkipVerify: true,
// }
func (wc *WebsocketClientBuilder) SetTLSConfig(c *tls.Config) *WebsocketClientBuilder {
	wc.tlsCfg = c
	return wc
}

// SetURL sets the target url.
// Example: ws://127.0.0.1:7878/hello/world
func (wc *WebsocketClientBuilder) SetURL(url string) *WebsocketClientBuilder {
	wc.url = url
	return wc
}

// SetHeader sets header.
func (wc *WebsocketClientBuilder) SetHeader(header http.Header) *WebsocketClientBuilder {
	wc.header = header
	return wc
}

// SetProxy sets proxy.
func (wc *WebsocketClientBuilder) SetProxy(proxy func(*http.Request) (*url.URL, error)) *WebsocketClientBuilder {
	wc.proxy = proxy
	return wc
}

// Build builds and returns a new websocket ClientTransporter
func (wc *WebsocketClientBuilder) Build() transport.ClientTransporter {
	return func(ctx context.Context) (*transport.Transport, error) {
		return transport.NewWebsocketClientTransport(ctx, wc.url, wc.tlsCfg, wc.header, wc.proxy)
	}
}

// SetHostAndPort sets the host and port.
func (ts *TCPServerBuilder) SetHostAndPort(host string, port int) *TCPServerBuilder {
	ts.addr = fmt.Sprintf("%s:%d", host, port)
	return ts
}

// SetAddr sets the addr.
func (ts *TCPServerBuilder) SetAddr(addr string) *TCPServerBuilder {
	ts.addr = addr
	return ts
}

// SetTLSConfig sets the tls config.
//
// You can generate cert.pem and key.pem for local testing:
//
//	 go run $GOROOT/src/crypto/tls/generate_cert.go --host localhost
//
//	 Load X509
//	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
//	if err != nil {
//		panic(err)
//	}
//	// Init TLS configuration.
//	tc := &tls.Config{
//		MinVersion:               tls.VersionTLS12,
//		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
//		PreferServerCipherSuites: true,
//		CipherSuites: []uint16{
//			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
//			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
//			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
//			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
//		},
//		Certificates: []tls.Certificate{cert},
//	}
func (ts *TCPServerBuilder) SetTLSConfig(c *tls.Config) *TCPServerBuilder {
	ts.tlsCfg = c
	return ts
}

// Build builds and returns a new TCP ServerTransporter.
func (ts *TCPServerBuilder) Build() transport.ServerTransporter {
	return func(ctx context.Context) (transport.ServerTransport, error) {
		return transport.NewTCPServerTransportWithAddr("tcp", ts.addr, ts.tlsCfg), nil
	}
}

// SetHostAndPort sets the host and port.
func (tc *TCPClientBuilder) SetHostAndPort(host string, port int) *TCPClientBuilder {
	tc.addr = fmt.Sprintf("%s:%d", host, port)
	return tc
}

// SetAddr sets the addr
func (tc *TCPClientBuilder) SetAddr(addr string) *TCPClientBuilder {
	tc.addr = addr
	return tc
}

// SetTLSConfig sets the tls config.
//
// Here's an example:
//
// tc := &tls.Config{
//	InsecureSkipVerify: true,
// }
func (tc *TCPClientBuilder) SetTLSConfig(c *tls.Config) *TCPClientBuilder {
	tc.tlsCfg = c
	return tc
}

// Build builds and returns a new TCP ClientTransporter.
func (tc *TCPClientBuilder) Build() transport.ClientTransporter {
	return func(ctx context.Context) (*transport.Transport, error) {
		return transport.NewTCPClientTransportWithAddr(ctx, "tcp", tc.addr, tc.tlsCfg)
	}
}

// TCPClient creates a new TCPClientBuilder
func TCPClient() *TCPClientBuilder {
	return &TCPClientBuilder{
		addr: fmt.Sprintf(":%d", DefaultPort),
	}
}

// TCPServer creates a new TCPServerBuilder
func TCPServer() *TCPServerBuilder {
	return &TCPServerBuilder{
		addr: fmt.Sprintf(":%d", DefaultPort),
	}
}

// WebsocketClient creates a new WebsocketClientBuilder.
func WebsocketClient() *WebsocketClientBuilder {
	return &WebsocketClientBuilder{
		url:   fmt.Sprintf("ws://127.0.0.1:%d", DefaultPort),
		proxy: http.ProxyFromEnvironment,
	}
}

// WebsocketServer creates a new WebsocketServerBuilder.
func WebsocketServer() *WebsocketServerBuilder {
	return &WebsocketServerBuilder{
		addr: fmt.Sprintf(":%d", DefaultPort),
		path: "/",
	}
}

// UnixClient creates a new UnixClientBuilder.
func UnixClient() *UnixClientBuilder {
	return &UnixClientBuilder{
		path: DefaultUnixSockPath,
	}
}

// UnixServer creates a new UnixServerBuilder.
func UnixServer() *UnixServerBuilder {
	return &UnixServerBuilder{
		path: DefaultUnixSockPath,
	}
}
