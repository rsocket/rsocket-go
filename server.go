package rsocket

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

func (p *Server) Close() error {
	return p.opts.transport.Close()
}

func (p *Server) Serve() error {
	p.opts.transport.Accept(func(setup *FrameSetup, conn RConnection) error {
		rs := newRSocket(conn)
		var v Version = [2]uint16{setup.Major(), setup.Minor()}
		sp := newSetupPayload(v, setup.Data(), setup.Metadata())
		return p.opts.acceptor(sp, rs)
	})
	return p.opts.transport.Listen()
}

type serverOptions struct {
	transport ServerTransport
	acceptor  Acceptor
}

type ServerOption func(o *serverOptions)

func WithTCPServerTransport(addr string) ServerOption {
	return func(o *serverOptions) {
		o.transport = newTCPServerTransport(addr, 0)
	}
}

func WithAcceptor(acceptor Acceptor) ServerOption {
	return func(o *serverOptions) {
		o.acceptor = acceptor
	}
}

func NewServer(opts ...ServerOption) (*Server, error) {
	o := &serverOptions{}
	for _, it := range opts {
		it(o)
	}
	if o.transport == nil {
		return nil, ErrInvalidTransport
	}
	return &Server{opts: o,}, nil
}
