package rsocket

import (
	"context"
	"errors"
)

var (
	errMissingTransport = errors.New("missing transport")
)

type Emitter interface {
	Next(payload Payload) error
	Complete(payload Payload) error
}

type Acceptor = func(setup SetupPayload, sendingSocket *RSocket) (err error)
type HandlerRQ = func(req Payload) (res Payload, err error)
type HandlerRS = func(req Payload, emitter Emitter)
type HandlerFNF = func(req Payload) error

type Server struct {
	opts *serverOptions
}

func (p *Server) Start(ctx context.Context) error {
	p.opts.transport.Accept(func(setup *FrameSetup, conn RConnection) error {
		rs := newRSocket(conn, p.opts.handlerRQ, p.opts.handlerRS, p.opts.handlerFNF)
		var v Version = [2]uint16{setup.Major(), setup.Minor()}
		sp := newSetupPayload(v, setup.Data(), setup.Metadata())
		return p.opts.acceptor(sp, rs)
	})
	return p.opts.transport.Listen(ctx)
}

type serverOptions struct {
	transport  ServerTransport
	acceptor   Acceptor
	handlerRQ  HandlerRQ
	handlerRS  HandlerRS
	handlerFNF HandlerFNF
}

type ServerOption func(o *serverOptions)

func WithTransportTCP(addr string) ServerOption {
	return func(o *serverOptions) {
		o.transport = newTCPServerTransport(addr, 0)
	}
}

func WithAcceptor(acceptor Acceptor) ServerOption {
	return func(o *serverOptions) {
		o.acceptor = acceptor
	}
}

func WithRequestResponseHandler(h HandlerRQ) ServerOption {
	return func(o *serverOptions) {
		o.handlerRQ = h
	}
}

func WithRequestStreamHandler(h HandlerRS) ServerOption {
	return func(o *serverOptions) {
		o.handlerRS = h
	}
}

func WithFireAndForget(h HandlerFNF) ServerOption {
	return func(o *serverOptions) {
		o.handlerFNF = h
	}
}

func NewServer(opts ...ServerOption) (*Server, error) {
	o := &serverOptions{}
	for _, it := range opts {
		it(o)
	}
	if o.transport == nil {
		return nil, errMissingTransport
	}
	return &Server{opts: o,}, nil
}
