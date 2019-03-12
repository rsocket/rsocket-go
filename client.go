package rsocket

import (
	"context"
	"io"
	"runtime"
	"time"
)

type ClientSocket interface {
	io.Closer
	RSocket
}

type ClientSocketAcceptor = func(socket RSocket) RSocket

type ClientStarter interface {
	Start() (ClientSocket, error)
}

type ClientBuilder interface {
	KeepAlive(tickPeriod, ackTimeout time.Duration, missedAcks int) ClientBuilder
	DataMimeType(mime string) ClientBuilder
	MetadataMimeType(mime string) ClientBuilder
	SetupPayload(setup Payload) ClientBuilder
	Transport(transport string) ClientStarter
	Acceptor(acceptor ClientSocketAcceptor) ClientTransportBuilder
}

type ClientTransportBuilder interface {
	Transport(transport string) ClientStarter
}

func Connect() ClientBuilder {
	return &implClientBuilder{
		keepaliveInteval:     defaultKeepaliveInteval,
		keepaliveMaxLifetime: defaultKeepaliveMaxLifetime,
		dataMimeType:         MimeTypeBinary,
		metadataMimeType:     MimeTypeBinary,
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

func (p *implClientBuilder) SetupPayload(setup Payload) ClientBuilder {
	defer setup.Release()

	p.setupData = nil
	p.setupMetadata = nil

	data := setup.Data()
	if len(data) > 0 {
		data2 := make([]byte, len(data))
		copy(data2, data)
		p.setupData = data2
	}
	metadata := setup.Metadata()
	if len(metadata) > 0 {
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
	tp, err := newClientTransportTCP(p.addr, p.keepaliveInteval, p.keepaliveMaxLifetime)
	if err != nil {
		return nil, err
	}
	sendingScheduler := NewElasticScheduler(runtime.NumCPU())
	tp.onClose(func() {
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
	setup := createSetup(defaultVersion, p.keepaliveInteval, p.keepaliveMaxLifetime, nil, p.metadataMimeType, p.dataMimeType, p.setupData, p.setupMetadata)
	if err := tp.Send(setup); err != nil {
		return nil, err
	}
	return requester, nil
}
