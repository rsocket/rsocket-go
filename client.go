package rsocket

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/fragmentation"
	"github.com/rsocket/rsocket-go/internal/transport"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/socket"
)

var defaultMimeType = []byte("application/binary")
var ErrClientClosed = errors.New("client has been closed")

type (
	// ClientSocket is Client Side of a RSocket socket. Sends Frames to a RSocket Server.
	Client interface {
		io.Closer
		RSocket
	}

	setupClientSocket interface {
		Client
		Setup(ctx context.Context, setup *socket.SetupInfo) error
		OnClose(fn func())
	}

	// ClientSocketAcceptor is alias for RSocket handler function.
	ClientSocketAcceptor = func(socket RSocket) RSocket

	// ClientStarter can be used to start a client.
	ClientStarter interface {
		// Start start a client socket.
		Start(ctx context.Context) (Client, error)
	}

	// ClientBuilder can be used to build a RSocket client.
	ClientBuilder interface {
		ClientTransportBuilder
		// Fragment set fragmentation size which default is 16_777_215(16MB).
		Fragment(mtu int) ClientBuilder
		// KeepAlive defines current client keepalive settings.
		KeepAlive(tickPeriod, ackTimeout time.Duration, missedAcks int) ClientBuilder
		// Resume enable resume for current RSocket.
		Resume(opts ...ResumeOption) ClientBuilder
		// DataMimeType is used to set payload data MIME type.
		// Default MIME type is `application/binary`.
		DataMimeType(mime string) ClientBuilder
		// MetadataMimeType is used to set payload metadata MIME type.
		// Default MIME type is `application/binary`.
		MetadataMimeType(mime string) ClientBuilder
		// SetupPayload set the setup payload.
		SetupPayload(setup payload.Payload) ClientBuilder
		// OnClose register handler when client socket closed.
		OnClose(fn func()) ClientBuilder
		// Acceptor set acceptor for RSocket client.
		Acceptor(acceptor ClientSocketAcceptor) ClientTransportBuilder

		// clone returns a copy of client builder.
		clone() ClientBuilder
	}

	// ClientTransportBuilder is used to build a RSocket client with custom Transport string.
	ClientTransportBuilder interface {
		// Transport set Transport for current RSocket client.
		// URI is used to create RSocket Transport:
		// Example:
		// "tcp://127.0.0.1:7878" means a TCP RSocket transport.
		// "ws://127.0.0.1:8080/a/b/c" means a Websocket RSocket transport. (NOTICE: Websocket will be supported in the future).
		Transport(uri string) ClientStarter
		// Transports set transports with load balancer.
		// Client will watch discovery and change current transports.
		// You can custom balancer options use functions: WithInitTransports, WithQuantile, WithPendings and WithActives.
		Transports(discovery <-chan []string, options ...OptBalancer) ClientStarter
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

type implClientBuilder struct {
	resume   *resumeOpts
	token    []byte
	fragment int
	addr     string
	setup    *socket.SetupInfo
	acceptor ClientSocketAcceptor
	onCloses []func()
}

func (p *implClientBuilder) Resume(opts ...ResumeOption) ClientBuilder {
	if p.resume == nil {
		p.resume = newResumeOpts()
	}
	for _, it := range opts {
		it(p.resume)
	}
	return p
}

func (p *implClientBuilder) Transports(discovery <-chan []string, options ...OptBalancer) ClientStarter {
	return newBalancerStarter(p, discovery, options...)
}

func (p *implClientBuilder) Fragment(mtu int) ClientBuilder {
	p.fragment = mtu
	return p
}

func (p *implClientBuilder) clone() ClientBuilder {
	clone := &implClientBuilder{
		fragment: p.fragment,
		setup:    p.setup,
		acceptor: p.acceptor,
	}
	if len(p.onCloses) > 0 {
		clone.onCloses = make([]func(), len(p.onCloses))
		copy(clone.onCloses, p.onCloses)
	}
	return clone
}

func (p *implClientBuilder) OnClose(fn func()) ClientBuilder {
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
	defer setup.Release()

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

func (p *implClientBuilder) Transport(transport string) ClientStarter {
	p.addr = transport
	return p
}

func (p *implClientBuilder) Start(ctx context.Context) (client Client, err error) {
	var uri *transport.URI
	uri, err = transport.ParseURI(p.addr)
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
		rx.ElasticScheduler(),
		p.setup.KeepaliveInterval,
	)

	// create a client.
	var cs setupClientSocket
	if p.resume != nil {
		p.setup.Token = p.resume.tokenGen()
		cs = socket.NewClientResume(uri, sk)
	} else {
		cs = socket.NewClient(uri, sk)
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

type ResumeOption func(opts *resumeOpts)

func ResumeToken(gen func() []byte) ResumeOption {
	return func(opts *resumeOpts) {
		opts.tokenGen = gen
	}
}
