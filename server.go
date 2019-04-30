package rsocket

import (
	"sync"

	"github.com/rsocket/rsocket-go/fragmentation"
	"github.com/rsocket/rsocket-go/framing"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/transport"
)

const serverWorkerPoolSize = 1000

type (
	// ServerBuilder can be used to build a RSocket server.
	ServerBuilder interface {
		// Fragment set fragmentation size which default is 16_777_215(16MB).
		Fragment(mtu int) ServerBuilder
		// Acceptor register server acceptor which is used to handle incoming RSockets.
		Acceptor(acceptor ServerAcceptor) ServerTransportBuilder
	}

	// ServerTransportBuilder is used to build a RSocket server with custom Transport string.
	ServerTransportBuilder interface {
		// Transport specify transport string.
		Transport(transport string) Start
	}

	// Start start a RSocket server.
	Start interface {
		// Serve serve RSocket server.
		Serve() error
	}
)

// Receive receives server connections from client RSockets.
func Receive() ServerBuilder {
	return &xServer{
		fragment:  fragmentation.MaxFragment,
		responses: &sync.Map{},
		scheduler: rx.NewElasticScheduler(serverWorkerPoolSize),
	}
}

type xServer struct {
	fragment  int
	addr      string
	acc       ServerAcceptor
	scheduler rx.Scheduler
	responses *sync.Map // sid -> flux/mono
}

func (p *xServer) Fragment(mtu int) ServerBuilder {
	p.fragment = mtu
	return p
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
	defer func() {
		_ = p.scheduler.Close()
	}()
	splitter, err := fragmentation.NewSplitter(p.fragment)
	if err != nil {
		return err
	}
	t, err := transport.NewTCPServerTransport(p.addr)
	if err != nil {
		return err
	}
	t.Accept(func(setup *framing.FrameSetup, tp transport.Transport) error {
		defer setup.Release()
		sendingSocket := newDuplexRSocket(tp, true, p.scheduler, splitter)
		sendingSocket.bindResponder(p.acc(setup, sendingSocket))
		return nil
	})
	return t.Listen()
}
