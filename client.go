package rsocket

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type Client struct {
	opts        *clientOpts
	c           RConnection
	sndStreamID uint32
	hReqRes     *sync.Map // map[uint32]
	wp          workerPool
	onceClose   *sync.Once
}

func (p *Client) Close() (err error) {
	p.onceClose.Do(func() {
		_ = p.wp.Close()
		if p.c != nil {
			err = p.c.Close()
			p.c = nil
		}
	})
	return
}

func (p *Client) FireAndForget(data, metadata []byte) error {
	sid := p.nextStreamID()
	f := createFNF(sid, data, metadata)
	return p.c.Send(f)
}

func (p *Client) MetadataPush(metadata []byte) error {
	return p.c.Send(createMetadataPush(metadata))
}

func (p *Client) RequestResponse(data, metadata []byte, handler func(res Payload, err error)) error {
	sid := p.nextStreamID()
	p.hReqRes.Store(sid, handler)
	err := p.c.Send(createRequestResponse(sid, data, metadata))
	if err != nil {
		p.hReqRes.Delete(sid)
	}
	return err
}

func (p *Client) Start(ctx context.Context) (err error) {
	keepaliveInterval := p.opts.tickPeriod
	keepaliveMaxLifetime := p.opts.ackTimeout * time.Duration(p.opts.missedAcks)
	p.c, err = p.opts.tp.Connect(keepaliveInterval)
	if err != nil {
		return
	}
	p.c.HandlePayload(func(f *framePayload) (err error) {
		sid := f.header.StreamID()
		value, ok := p.hReqRes.Load(sid)
		if !ok {
			defer f.Release()
			return
		}
		fn := value.(func(Payload, error))
		p.hReqRes.Delete(sid)
		p.wp.Do(func() {
			defer f.Release()
			fn(f, nil)
		})
		return
	})
	p.c.PostFlight(ctx)
	setup := createSetup(defaultVersion, keepaliveInterval, keepaliveMaxLifetime, nil, p.opts.mimeMetadata, p.opts.mimeData, p.opts.setupMetadata, p.opts.setupData)
	err = p.c.Send(setup)
	return
}

func (p *Client) nextStreamID() uint32 {
	return 2*(atomic.AddUint32(&p.sndStreamID, 1)-1) + 1
}

type clientOpts struct {
	tp            Transport
	setupData     []byte
	setupMetadata []byte

	tickPeriod time.Duration
	ackTimeout time.Duration
	missedAcks int

	mimeData     []byte
	mimeMetadata []byte

	workerPoolSize int
}

var (
	MimeTypeBinary = []byte("application/binary")
)

type ClientOption func(o *clientOpts)

func NewClient(options ...ClientOption) (*Client, error) {
	o := &clientOpts{
	}
	for _, it := range options {
		it(o)
	}
	if o.tp == nil {
		return nil, ErrInvalidTransport
	}
	if o.mimeMetadata == nil {
		o.mimeMetadata = MimeTypeBinary
	}
	if o.mimeData == nil {
		o.mimeData = MimeTypeBinary
	}

	if o.tickPeriod < 1 {
		o.tickPeriod = 20 * time.Second
	}
	if o.ackTimeout < 1 {
		o.ackTimeout = 30 * time.Second
	}
	if o.missedAcks < 1 {
		o.missedAcks = 3
	}
	return &Client{
		opts:      o,
		hReqRes:   &sync.Map{},
		wp:        newWorkerPool(o.workerPoolSize),
		onceClose: &sync.Once{},
	}, nil
}

func WithClientWorkerPoolSize(n int) ClientOption {
	return func(o *clientOpts) {
		o.workerPoolSize = n
	}
}

func WithTCPTransport(host string, port int) ClientOption {
	return func(o *clientOpts) {
		o.tp = newTCPClientTransport(host, port)
	}
}

func WithSetupPayload(data []byte, metadata []byte) ClientOption {
	return func(o *clientOpts) {
		o.setupData = data
		o.setupMetadata = metadata
	}
}

func WithKeepalive(tickPeriod time.Duration, ackTimeout time.Duration, missedAcks int) ClientOption {
	if tickPeriod < 1 {
		panic(fmt.Errorf("invalid tickPeriod: %d", tickPeriod))
	}
	if ackTimeout < 1 {
		panic(fmt.Errorf("invalid ackTimeout: %d", ackTimeout))
	}
	if missedAcks < 1 {
		panic(fmt.Errorf("invalid missedAcks: %d", missedAcks))
	}
	return func(o *clientOpts) {
		o.tickPeriod = tickPeriod
		o.ackTimeout = ackTimeout
		o.missedAcks = missedAcks
	}
}

func WithDataMimeType(mime string) ClientOption {
	return func(o *clientOpts) {
		o.mimeData = []byte(mime)
	}
}

func WithMetadataMimeType(mime string) ClientOption {
	return func(o *clientOpts) {
		o.mimeMetadata = []byte(mime)
	}
}
