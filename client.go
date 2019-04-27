package rsocket

import (
	"context"
	"io"
	"runtime"
	"time"

	"github.com/rsocket/rsocket-go/common"
	"github.com/rsocket/rsocket-go/common/logger"
	"github.com/rsocket/rsocket-go/fragmentation"
	"github.com/rsocket/rsocket-go/framing"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/transport"
)

var defaultMimeType = []byte("application/binary")

type (
	// ClientSocket is Client Side of v RSocket socket. Sends Frames to v RSocket Server.
	ClientSocket interface {
		io.Closer
		RSocket
	}

	// ClientSocketAcceptor is alias for RSocket handler function.
	ClientSocketAcceptor = func(socket RSocket) RSocket

	// ClientStarter can be used to start v client.
	ClientStarter interface {
		// Start start v client socket.
		Start() (ClientSocket, error)
	}

	// ClientBuilder can be used to build v RSocket client.
	ClientBuilder interface {
		ClientTransportBuilder
		// Fragment set fragmentation size which default is 16_777_215(16MB).
		Fragment(mtu int) ClientBuilder
		// KeepAlive defines current client keepalive settings.
		KeepAlive(tickPeriod, ackTimeout time.Duration, missedAcks int) ClientBuilder
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

	// ClientTransportBuilder is used to build v RSocket client with custom Transport string.
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

// Connect create v new RSocket client builder with default settings.
func Connect() ClientBuilder {
	return &implClientBuilder{
		fragment:             fragmentation.MaxFragment,
		keepaliveInteval:     common.DefaultKeepaliveInteval,
		keepaliveMaxLifetime: common.DefaultKeepaliveMaxLifetime,
		dataMimeType:         defaultMimeType,
		metadataMimeType:     defaultMimeType,
		onCloses:             make([]func(), 0),
	}
}

type implClientBuilder struct {
	fragment             int
	addr                 string
	keepaliveInteval     time.Duration
	keepaliveMaxLifetime time.Duration
	dataMimeType         []byte
	metadataMimeType     []byte
	setupData            []byte
	setupMetadata        []byte
	acceptor             ClientSocketAcceptor
	onCloses             []func()
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
		fragment:             p.fragment,
		keepaliveInteval:     p.keepaliveInteval,
		keepaliveMaxLifetime: p.keepaliveMaxLifetime,
		dataMimeType:         p.dataMimeType,
		metadataMimeType:     p.metadataMimeType,
		setupData:            p.setupData,
		setupMetadata:        p.setupMetadata,
		acceptor:             p.acceptor,
		onCloses:             make([]func(), len(p.onCloses)),
	}
	copy(clone.onCloses, p.onCloses)
	return clone
}

func (p *implClientBuilder) OnClose(fn func()) ClientBuilder {
	p.onCloses = append(p.onCloses, fn)
	return p
}

func (p *implClientBuilder) KeepAlive(tickPeriod, ackTimeout time.Duration, missedAcks int) ClientBuilder {
	p.keepaliveInteval = tickPeriod
	p.keepaliveMaxLifetime = time.Duration(missedAcks) * ackTimeout
	return p
}

func (p *implClientBuilder) DataMimeType(mime string) ClientBuilder {
	p.dataMimeType = []byte(mime)
	return p
}

func (p *implClientBuilder) MetadataMimeType(mime string) ClientBuilder {
	p.metadataMimeType = []byte(mime)
	return p
}

func (p *implClientBuilder) SetupPayload(setup payload.Payload) ClientBuilder {
	defer setup.Release()

	p.setupData = nil
	p.setupMetadata = nil

	data := setup.Data()
	if len(data) > 0 {
		data2 := make([]byte, len(data))
		copy(data2, data)
		p.setupData = data2
	}
	if metadata, ok := setup.Metadata(); ok {
		metadata2 := make([]byte, len(metadata))
		copy(metadata2, metadata)
		p.setupMetadata = metadata2
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

func (p *implClientBuilder) Start() (ClientSocket, error) {
	tpURI, err := transport.ParseURI(p.addr)
	if err != nil {
		return nil, err
	}
	splitter, err := fragmentation.NewSplitter(p.fragment)
	if err != nil {
		return nil, err
	}
	tp, err := tpURI.MakeClientTransport(p.keepaliveInteval, p.keepaliveMaxLifetime)
	if err != nil {
		return nil, err
	}
	sendingScheduler := rx.NewElasticScheduler(runtime.NumCPU())
	tp.OnClose(func() {
		_ = sendingScheduler.Close()
	})
	for _, it := range p.onCloses {
		tp.OnClose(it)
	}
	requester := newDuplexRSocket(tp, false, sendingScheduler, splitter)
	if p.acceptor != nil {
		requester.bindResponder(p.acceptor(requester))
	}
	go func(ctx context.Context) {
		if err := tp.Start(ctx); err != nil {
			logger.Debugf("client exit: %s\n", err)
		}
	}(context.Background())
	setup := framing.NewFrameSetup(common.DefaultVersion, p.keepaliveInteval, p.keepaliveMaxLifetime, nil, p.metadataMimeType, p.dataMimeType, p.setupData, p.setupMetadata)
	if err := tp.Send(setup); err != nil {
		return nil, err
	}
	return requester, nil
}
