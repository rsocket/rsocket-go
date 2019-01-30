package rsocket

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type Client struct {
	opts        *clientOpts
	c           RConnection
	sndStreamID uint32
	hReqRes     *sync.Map //map[uint32]
	fragmenter  Fragmenter
}

func (p *Client) Close() error {
	if p.c == nil {
		return nil
	}
	return p.c.Close()
}

func (p *Client) RequestResponse(req Payload, handler func(res Payload, err error)) error {
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
	if err := p.c.Send(mkRequestResponse(sid, req.Metadata(), req.Data(), fg)); err != nil {
		return err
	}
	p.hReqRes.Store(sid, handler)
	return nil
}

func (p *Client) Start(ctx context.Context) (err error) {
	p.c, err = p.opts.tp.Connect()
	if err != nil {
		return
	}

	p.c.HandlePayload(func(f *FramePayload) (err error) {
		frame := p.fragmenter.Fragment(f)
		if frame == nil {
			return nil
		}
		sid := frame.StreamID()
		if value, ok := p.hReqRes.Load(sid); ok {
			fn := value.(func(Payload, error))
			p.hReqRes.Delete(sid)
			payload := CreatePayloadRaw(frame.data, frame.metadata)
			fn(payload, nil)
		}
		return nil
	})

	p.c.PostFlight(ctx)
	setup := mkSetup(p.opts.setupMetadata, p.opts.setupData, p.opts.mimeMetadata, p.opts.mimeData, FlagMetadata)
	err = p.c.Send(setup)
	return
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
}

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
	return &Client{
		opts:    o,
		hReqRes: &sync.Map{},
	}, nil
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
