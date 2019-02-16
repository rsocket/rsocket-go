package rsocket

import (
	"context"
	"io"
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
		keepaliveInteval:     20 * time.Second,
		keepaliveMaxLifetime: 90 * time.Second,
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
	tp, err := newClientTransportTCP(p.addr)
	if err != nil {
		return nil, err
	}
	tp.onClose(func() {
		logger.Infof("client transport closed!!!!!\n")
	})
	go func(ctx context.Context) {
		_ = tp.Start(ctx)
	}(context.Background())
	setup := createSetup(defaultVersion, p.keepaliveInteval, p.keepaliveMaxLifetime, nil, p.metadataMimeType, p.dataMimeType, p.setupData, p.setupMetadata)
	if err := tp.Send(setup); err != nil {
		defer func() {
			_ = tp.Close()
		}()
		return nil, err
	}
	requester := newDuplexRSocket(tp, false)
	if p.acceptor != nil {
		responder := p.acceptor(requester)
		requester.bindResponder(responder)
	}

	return requester, nil
}
