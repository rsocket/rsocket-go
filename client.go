package rsocket

import (
	"context"
	"fmt"
	"log"
	"sync/atomic"
	"time"
)

type Client struct {
	opts        *clientOpts
	c           RConnection
	sndStreamID uint32

	hReqRes map[uint32]func(Payload)
}

func (p *Client) Close() error {
	return p.c.Close()
}

func (p *Client) RequestResponse(req Payload, handler func(res Payload)) error {
	var sid uint32
	if atomic.CompareAndSwapUint32(&p.sndStreamID, 0, 1) {
		sid = 1
	} else {
		sid = atomic.AddUint32(&p.sndStreamID, 2)
	}
	var fg Flags
	if req.Metadata() != nil {
		fg |= FlagMetadata
	}
	err := p.c.Send(mkRequestResponse(sid, req.Metadata(), req.Data(), fg))
	if err != nil {
		return err
	}
	p.hReqRes[sid] = handler
	return nil
}

func (p *Client) Start(ctx context.Context) (err error) {
	p.c, err = p.opts.tp.Connect(fmt.Sprintf("%s:%d", p.opts.host, p.opts.port))
	if err != nil {
		return
	}
	defer func() {
		if err := p.Close(); err != nil {
			log.Println("close client failed:", err)
		}
	}()

	p.c.HandlePayload(func(frame *FramePayload) (err error) {
		sid := frame.StreamID()
		if h, ok := p.hReqRes[sid]; ok {
			delete(p.hReqRes, sid)
			h(CreatePayloadRaw(frame.data, frame.metadata))
		}
		return nil
	})

	p.c.PostFlight(ctx)
	setup := mkSetup(p.opts.setupMetadata, p.opts.setupData, p.opts.mimeMetadata, p.opts.mimeData, FlagMetadata)
	err = p.c.Send(setup)
	return
}

type clientOpts struct {
	host          string
	port          int
	tp            Transport
	setupData     []byte
	setupMetadata []byte

	tickPeriod time.Duration
	ackTimeout time.Duration
	missedAcks int

	mimeData     string
	mimeMetadata string
}

type ClientOption func(o *clientOpts)

func NewClient(host string, port int, options ...ClientOption) (*Client, error) {
	o := &clientOpts{
		host: host,
		port: port,
	}
	for _, it := range options {
		it(o)
	}
	if o.tp == nil {
		return nil, errMissingTransport
	}
	return &Client{
		opts: o,
	}, nil
}

func WithSetupPayload(data []byte, metadata []byte) ClientOption {
	return func(o *clientOpts) {
		o.setupData = data
		o.setupMetadata = metadata
	}
}

func WithKeepalive(tickPeriod time.Duration, ackTimeout time.Duration, missedAcks int) ClientOption {
	return func(o *clientOpts) {
		o.tickPeriod = tickPeriod
		o.ackTimeout = ackTimeout
		o.missedAcks = missedAcks
	}
}

func WithDataMimeType(mime string) ClientOption {
	return func(o *clientOpts) {
		o.mimeData = mime
	}
}

func WithMetadataMimeType(mime string) ClientOption {
	return func(o *clientOpts) {
		o.mimeMetadata = mime
	}
}
