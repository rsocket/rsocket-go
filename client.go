package rsocket

import (
	"context"
	"github.com/rsocket/rsocket-go/common"
	"github.com/rsocket/rsocket-go/common/logger"
	"github.com/rsocket/rsocket-go/framing"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/transport"
	"io"
	"runtime"
	"time"
)

var defaultMimeType = []byte("application/binary")

type (
	// ClientSocket is Client Side of a RSocket socket. Sends Frames to a RSocket Server.
	ClientSocket interface {
		io.Closer
		RSocket
	}

	// ClientSocketAcceptor is alias for RSocket handler function.
	ClientSocketAcceptor = func(socket RSocket) RSocket

	// ClientStarter can be used to start a client.
	ClientStarter interface {
		// Start start a client socket.
		Start() (ClientSocket, error)
	}

	// ClientBuilder can be used to build a RSocket client.
	ClientBuilder interface {
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
		// Transport set Transport for current RSocket client.
		// In current version, you can use `$HOST:$PORT` string as TCP Transport.
		Transport(transport string) ClientStarter
		// Acceptor set acceptor for RSocket client.
		Acceptor(acceptor ClientSocketAcceptor) ClientTransportBuilder
	}

	// ClientTransportBuilder is used to build a RSocket client with custom Transport string.
	ClientTransportBuilder interface {
		// Transport set Transport string.
		Transport(transport string) ClientStarter
	}
)

// Connect create a new RSocket client builder with default settings.
func Connect() ClientBuilder {
	return &implClientBuilder{
		keepaliveInteval:     common.DefaultKeepaliveInteval,
		keepaliveMaxLifetime: common.DefaultKeepaliveMaxLifetime,
		dataMimeType:         defaultMimeType,
		metadataMimeType:     defaultMimeType,
	}
}

type implClientBuilder struct {
	addr                 string
	keepaliveInteval     time.Duration
	keepaliveMaxLifetime time.Duration
	dataMimeType         []byte
	metadataMimeType     []byte
	setupData            []byte
	setupMetadata        []byte
	acceptor             ClientSocketAcceptor
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
	tp, err := transport.NewClientTransportTCP(p.addr, p.keepaliveInteval, p.keepaliveMaxLifetime)
	if err != nil {
		return nil, err
	}
	sendingScheduler := rx.NewElasticScheduler(runtime.NumCPU())
	tp.OnClose(func() {
		_ = sendingScheduler.Close()
	})
	requester := newDuplexRSocket(tp, false, sendingScheduler)
	if p.acceptor != nil {
		requester.bindResponder(p.acceptor(requester))
	}
	go func(ctx context.Context) {
		if err := tp.Start(ctx); err != nil {
			logger.Debugf("client closed: %s\n", err)
		}
	}(context.Background())
	setup := framing.NewFrameSetup(common.DefaultVersion, p.keepaliveInteval, p.keepaliveMaxLifetime, nil, p.metadataMimeType, p.dataMimeType, p.setupData, p.setupMetadata)
	if err := tp.Send(setup); err != nil {
		return nil, err
	}
	return requester, nil
}
