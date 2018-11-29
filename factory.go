package rsocket

import (
	"errors"
	"github.com/jjeffcaii/go-rsocket/protocol"
)

var errMissingTransport = errors.New("missing transport")

type serverOptions struct {
	transport protocol.ServerTransport
	acceptor  Acceptor
	handlerRQ HandlerRQ
}

type ServerOption func(o *serverOptions)

func WithTransportTCP(addr string) ServerOption {
	return func(o *serverOptions) {
		o.transport = protocol.NewTcpServerTransport(addr)
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

func NewServer(opts ...ServerOption) (*Server, error) {
	o := &serverOptions{}
	for _, it := range opts {
		it(o)
	}
	if o.transport == nil {
		return nil, errMissingTransport
	}
	return &Server{
		transport: o.transport,
		acceptor:  o.acceptor,
		handlerRQ: o.handlerRQ,
	}, nil
}
