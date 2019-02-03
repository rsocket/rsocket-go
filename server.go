package rsocket

type Acceptor = func(setup SetupPayload, sendingSocket *RSocket) (err error)
type HandlerRQ = func(req Payload) (res Payload, err error)
type HandlerRS = func(req Payload) (res Flux)
type HandlerRC = func(req Flux) (res Flux)
type HandlerFNF = func(req Payload)
type HandlerMetadataPush = func(metadata []byte)

type Server struct {
	opts *serverOptions
}

func (p *Server) Close() error {
	return p.opts.transport.Close()
}

func (p *Server) Serve() error {
	wp := newWorkerPool(p.opts.workerPoolSize)
	defer func() {
		_ = wp.Close()
	}()
	p.opts.transport.Accept(func(setup *frameSetup, conn RConnection) error {
		rs := newRSocket(conn, wp)
		return p.opts.acceptor(setup, rs)
	})
	return p.opts.transport.Listen()
}

type serverOptions struct {
	workerPoolSize int
	transport      ServerTransport
	acceptor       Acceptor
}

type ServerOption func(o *serverOptions)

func WithServerWorkerPoolSize(n int) ServerOption {
	return func(o *serverOptions) {
		o.workerPoolSize = n
	}
}

func WithTCPServerTransport(addr string) ServerOption {
	return func(o *serverOptions) {
		o.transport = newTCPServerTransport(addr)
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
	return &Server{opts: o}, nil
}
