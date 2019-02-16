package rsocket

import (
	"sync"
)

type ServerBuilder interface {
	Acceptor(acceptor ServerAcceptor) ServerTransportBuilder
}

type ServerTransportBuilder interface {
	Transport(transport string) Start
}

type Start interface {
	Serve() error
}

type xServer struct {
	addr string
	acc  ServerAcceptor

	responses *sync.Map // sid -> flux/mono
}

func (p *xServer) Acceptor(acceptor ServerAcceptor) ServerTransportBuilder {
	p.acc = acceptor
	return p
}

func (p *xServer) Transport(transport string) Start {
	p.addr = transport
	return p
}

func (p *xServer) Serve() error {
	t := newTCPServerTransport(p.addr)
	t.Accept(func(setup *frameSetup, tp transport) error {
		defer setup.Release()
		sendingSocket := newDuplexRSocket(tp, true)
		socket := p.acc(setup, sendingSocket)
		sendingSocket.bindResponder(socket)
		return nil
	})
	return t.Listen()
}

func Receive() ServerBuilder {
	return &xServer{
		responses: &sync.Map{},
	}
}
