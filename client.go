package rsocket

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/google/uuid"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/fragmentation"
	"github.com/rsocket/rsocket-go/internal/socket"
	"github.com/rsocket/rsocket-go/internal/transport"
	"github.com/rsocket/rsocket-go/payload"
)

var defaultMimeType = []byte("application/binary")

type (
	// ClientResumeOptions represents resume options for client.
	ClientResumeOptions func(opts *resumeOpts)

	// Client is Client Side of a RSocket socket. Sends Frames to a RSocket Server.
	Client interface {
		CloseableRSocket
	}

	setupClientSocket interface {
		Client
		Setup(ctx context.Context, setup *socket.SetupInfo) error
	}

	// ClientSocketAcceptor is alias for RSocket handler function.
	ClientSocketAcceptor = func(socket RSocket) RSocket

	// ClientStarter can be used to start a client.
	ClientStarter interface {
		// Start start a client socket.
		Start(ctx context.Context) (Client, error)
		// Start start a client socket with TLS.
		// Here's an example:
		// tc:=&tls.Config{
		//	InsecureSkipVerify: true,
		//}
		StartTLS(ctx context.Context, tc *tls.Config) (Client, error)
	}

	// ClientBuilder can be used to build a RSocket client.
	ClientBuilder interface {
		ClientTransportBuilder
		// Fragment set fragmentation size which default is 16_777_215(16MB).
		Fragment(mtu int) ClientBuilder

		// KeepAlive defines current client keepalive settings.
		KeepAlive(tickPeriod, ackTimeout time.Duration, missedAcks int) ClientBuilder
		// Resume enable resume for current RSocket.
		Resume(opts ...ClientResumeOptions) ClientBuilder
		Lease() ClientBuilder
		// DataMimeType is used to set payload data MIME type.
		// Default MIME type is `application/binary`.
		DataMimeType(mime string) ClientBuilder
		// MetadataMimeType is used to set payload metadata MIME type.
		// Default MIME type is `application/binary`.
		MetadataMimeType(mime string) ClientBuilder
		// SetupPayload set the setup payload.
		SetupPayload(setup payload.Payload) ClientBuilder
		// OnClose register handler when client socket closed.
		OnClose(fn func(error)) ClientBuilder
		// Acceptor set acceptor for RSocket client.
		Acceptor(acceptor ClientSocketAcceptor) ClientTransportBuilder
	}

	// ClientTransportBuilder is used to build a RSocket client with custom Transport string.
	ClientTransportBuilder interface {
		// Transport set Transport for current RSocket client.
		// URI is used to create RSocket Transport:
		// Example:
		// "tcp://127.0.0.1:7878" means a TCP RSocket transport.
		// "ws://127.0.0.1:8080/a/b/c" means a Websocket RSocket transport.
		// "wss://127.0.0.1:8080/a/b/c" means a  Websocket RSocket transport with HTTPS.
		Transport(uri string, opts ...TransportOpts) ClientStarter
	}
)

// Connect create a new RSocket client builder with default settings.
func Connect() ClientBuilder {
	return &implClientBuilder{
		fragment: fragmentation.MaxFragment,
		setup: &socket.SetupInfo{
			Version:           common.DefaultVersion,
			KeepaliveInterval: common.DefaultKeepaliveInteval,
			KeepaliveLifetime: common.DefaultKeepaliveMaxLifetime,
			DataMimeType:      defaultMimeType,
			MetadataMimeType:  defaultMimeType,
		},
	}
}

type transportOpts struct {
	addr    string
	headers map[string][]string
}

// WithWebsocketHeaders attach headers for websocket transport.
func WithWebsocketHeaders(headers map[string][]string) TransportOpts {
	return func(opts *transportOpts) {
		opts.headers = headers
	}
}

// TransportOpts represents options of transport.
type TransportOpts = func(*transportOpts)

type implClientBuilder struct {
	resume   *resumeOpts
	fragment int
	tpOpts   *transportOpts
	setup    *socket.SetupInfo
	acceptor ClientSocketAcceptor
	onCloses []func(error)
}

func (p *implClientBuilder) Lease() ClientBuilder {
	p.setup.Lease = true
	return p
}

func (p *implClientBuilder) Resume(opts ...ClientResumeOptions) ClientBuilder {
	if p.resume == nil {
		p.resume = newResumeOpts()
	}
	for _, it := range opts {
		it(p.resume)
	}
	return p
}

func (p *implClientBuilder) Fragment(mtu int) ClientBuilder {
	p.fragment = mtu
	return p
}

func (p *implClientBuilder) OnClose(fn func(error)) ClientBuilder {
	p.onCloses = append(p.onCloses, fn)
	return p
}

func (p *implClientBuilder) KeepAlive(tickPeriod, ackTimeout time.Duration, missedAcks int) ClientBuilder {
	p.setup.KeepaliveInterval = tickPeriod
	p.setup.KeepaliveLifetime = time.Duration(missedAcks) * ackTimeout
	return p
}

func (p *implClientBuilder) DataMimeType(mime string) ClientBuilder {
	p.setup.DataMimeType = []byte(mime)
	return p
}

func (p *implClientBuilder) MetadataMimeType(mime string) ClientBuilder {
	p.setup.MetadataMimeType = []byte(mime)
	return p
}

func (p *implClientBuilder) SetupPayload(setup payload.Payload) ClientBuilder {
	p.setup.Data = nil
	p.setup.Metadata = nil

	if data := setup.Data(); len(data) > 0 {
		p.setup.Data = make([]byte, len(data))
		copy(p.setup.Data, data)
	}
	if metadata, ok := setup.Metadata(); ok {
		p.setup.Metadata = make([]byte, len(metadata))
		copy(p.setup.Metadata, metadata)
	}
	return p
}

func (p *implClientBuilder) Acceptor(acceptor ClientSocketAcceptor) ClientTransportBuilder {
	p.acceptor = acceptor
	return p
}

func (p *implClientBuilder) Transport(transport string, opts ...TransportOpts) ClientStarter {
	p.tpOpts = &transportOpts{
		addr: transport,
	}
	for i := 0; i < len(opts); i++ {
		opts[i](p.tpOpts)
	}
	return p
}

func (p *implClientBuilder) StartTLS(ctx context.Context, tc *tls.Config) (Client, error) {
	return p.start(ctx, tc)
}

func (p *implClientBuilder) Start(ctx context.Context) (client Client, err error) {
	return p.start(ctx, nil)
}

func (p *implClientBuilder) start(ctx context.Context, tc *tls.Config) (client Client, err error) {
	var uri *transport.URI
	uri, err = transport.ParseURI(p.tpOpts.addr)
	if err != nil {
		return
	}

	// create a blank socket.
	err = fragmentation.IsValidFragment(p.fragment)
	if err != nil {
		return nil, err
	}

	sk := socket.NewClientDuplexRSocket(
		p.fragment,
		p.setup.KeepaliveInterval,
	)
	var headers map[string][]string
	if uri.IsWebsocket() {
		headers = p.tpOpts.headers
	}
	// create a client.
	var cs setupClientSocket
	if p.resume != nil {
		p.setup.Token = p.resume.tokenGen()
		cs = socket.NewClientResume(uri, sk, tc, headers)
	} else {
		cs = socket.NewClient(uri, sk, tc, headers)
	}
	if p.acceptor != nil {
		sk.SetResponder(p.acceptor(cs))
	}

	// bind closers.
	if len(p.onCloses) > 0 {
		for _, closer := range p.onCloses {
			cs.OnClose(closer)
		}
	}

	// setup client.
	err = cs.Setup(ctx, p.setup)
	if err == nil {
		client = cs
	}
	return
}

type resumeOpts struct {
	tokenGen func() []byte
}

func newResumeOpts() *resumeOpts {
	return &resumeOpts{
		tokenGen: getPresetResumeTokenGen,
	}
}

func getPresetResumeTokenGen() (token []byte) {
	token, _ = uuid.New().MarshalBinary()
	return
}

// WithClientResumeToken creates a resume token generator.
func WithClientResumeToken(gen func() []byte) ClientResumeOptions {
	return func(opts *resumeOpts) {
		opts.tokenGen = gen
	}
}
