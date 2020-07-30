package rsocket

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/fragmentation"
	"github.com/rsocket/rsocket-go/internal/socket"
	"github.com/rsocket/rsocket-go/payload"
)

var (
	_defaultMimeType = []byte("application/binary")
	_noopSocket      = NewAbstractSocket()
)

type (
	// ClientResumeOptions represents resume options for client.
	ClientResumeOptions func(opts *resumeOpts)
)

// Client is Client Side of a RSocket socket. Sends Frames to a RSocket Server.
type Client interface {
	CloseableRSocket
}

// ClientSocketAcceptor is alias for RSocket handler function.
type ClientSocketAcceptor = func(socket RSocket) RSocket

// ClientStarter can be used to start a client.
type ClientStarter interface {
	// Start start a client socket.
	Start(ctx context.Context) (Client, error)
}

// ClientBuilder can be used to build a RSocket client.
type ClientBuilder interface {
	ToClientStarter
	// Fragment set fragmentation size which default is 16_777_215(16MB).
	// Also zero mtu means using default fragmentation size.
	Fragment(mtu int) ClientBuilder
	// KeepAlive defines current client keepalive settings.
	KeepAlive(tickPeriod, ackTimeout time.Duration, missedAcks int) ClientBuilder
	// Resume enable the functionality of resume.
	Resume(opts ...ClientResumeOptions) ClientBuilder
	// Lease enable the functionality of lease.
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
	Acceptor(acceptor ClientSocketAcceptor) ToClientStarter
}

type ToClientStarter interface {
	// Transport set Transport for current RSocket client.
	// URI is used to create RSocket Transport:
	// Example:
	// "tcp://127.0.0.1:7878" means a TCP RSocket transport.
	// "ws://127.0.0.1:8080/a/b/c" means a Websocket RSocket transport.
	// "wss://127.0.0.1:8080/a/b/c" means a  Websocket RSocket transport with HTTPS.
	Transport(Transporter) ClientStarter
}

// ToClientStarter is used to build a RSocket client with custom Transport string.
type setupClientSocket interface {
	Client
	Setup(ctx context.Context, setup *socket.SetupInfo) error
}

type clientBuilder struct {
	resume   *resumeOpts
	fragment int
	tpGen    transport.ClientTransportFunc
	setup    *socket.SetupInfo
	acceptor ClientSocketAcceptor
	onCloses []func(error)
}

func (p *clientBuilder) Lease() ClientBuilder {
	p.setup.Lease = true
	return p
}

func (p *clientBuilder) Resume(opts ...ClientResumeOptions) ClientBuilder {
	if p.resume == nil {
		p.resume = newResumeOpts()
	}
	for _, it := range opts {
		it(p.resume)
	}
	return p
}

func (p *clientBuilder) Fragment(mtu int) ClientBuilder {
	if mtu == 0 {
		p.fragment = fragmentation.MaxFragment
	} else {
		p.fragment = mtu
	}
	return p
}

func (p *clientBuilder) OnClose(fn func(error)) ClientBuilder {
	p.onCloses = append(p.onCloses, fn)
	return p
}

func (p *clientBuilder) KeepAlive(tickPeriod, ackTimeout time.Duration, missedAcks int) ClientBuilder {
	p.setup.KeepaliveInterval = tickPeriod
	p.setup.KeepaliveLifetime = time.Duration(missedAcks) * ackTimeout
	return p
}

func (p *clientBuilder) DataMimeType(mime string) ClientBuilder {
	p.setup.DataMimeType = []byte(mime)
	return p
}

func (p *clientBuilder) MetadataMimeType(mime string) ClientBuilder {
	p.setup.MetadataMimeType = []byte(mime)
	return p
}

func (p *clientBuilder) SetupPayload(setup payload.Payload) ClientBuilder {
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

func (p *clientBuilder) Acceptor(acceptor ClientSocketAcceptor) ToClientStarter {
	p.acceptor = acceptor
	return p
}

func (p *clientBuilder) Transport(support Transporter) ClientStarter {
	p.tpGen = support.Client()
	return p
}

func (p *clientBuilder) Start(ctx context.Context) (client Client, err error) {
	// create a blank socket.
	err = fragmentation.IsValidFragment(p.fragment)
	if err != nil {
		return nil, err
	}

	sk := socket.NewClientDuplexConnection(
		p.fragment,
		p.setup.KeepaliveInterval,
	)
	// create a client.
	var cs setupClientSocket
	if p.resume != nil {
		p.setup.Token = p.resume.tokenGen()
		cs = socket.NewResumableClientSocket(p.tpGen, sk)
	} else {
		cs = socket.NewClient(p.tpGen, sk)
	}
	if p.acceptor != nil {
		sk.SetResponder(p.acceptor(cs))
	} else {
		sk.SetResponder(_noopSocket)
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

// Connect create a new RSocket client builder with default settings.
func Connect() ClientBuilder {
	return &clientBuilder{
		fragment: fragmentation.MaxFragment,
		setup: &socket.SetupInfo{
			Version:           core.DefaultVersion,
			KeepaliveInterval: common.DefaultKeepaliveInterval,
			KeepaliveLifetime: common.DefaultKeepaliveMaxLifetime,
			DataMimeType:      _defaultMimeType,
			MetadataMimeType:  _defaultMimeType,
		},
	}
}
