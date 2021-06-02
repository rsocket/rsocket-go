package rsocket

import (
	"context"
	"github.com/rsocket/rsocket-go/internal/bytesconv"
	"time"

	"github.com/google/uuid"
	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/fragmentation"
	"github.com/rsocket/rsocket-go/internal/socket"
	"github.com/rsocket/rsocket-go/payload"
)

var (
	_defaultMimeType = []byte("application/binary")
	_noopSocket      = NewAbstractSocket()
)

type (
	// ClientResumeOptions represents resume options for client.
	ClientResumeOptions func(opts *resumeOpts)
)

// Client is Client Side of a RSocket socket. Sends Frames to a RSocket Server.
type Client interface {
	CloseableRSocket
}

// ClientSocketAcceptor is alias for RSocket handler function.
type ClientSocketAcceptor = func(ctx context.Context, socket RSocket) RSocket

// ClientStarter can be used to start a client.
type ClientStarter interface {
	// Start start a client socket.
	Start(ctx context.Context) (Client, error)
}

// ClientBuilder can be used to build a RSocket client.
type ClientBuilder interface {
	ToClientStarter
	// Scheduler set schedulers for the requests or responses.
	// Nil scheduler means keep the default scheduler settings.
	Scheduler(requestScheduler scheduler.Scheduler, responseScheduler scheduler.Scheduler) ClientBuilder
	// Fragment set fragmentation size which default is 16_777_215(16MB).
	// Also zero mtu means using default fragmentation size.
	Fragment(mtu int) ClientBuilder
	// KeepAlive defines current client keepalive settings.
	KeepAlive(tickPeriod, ackTimeout time.Duration, missedAcks int) ClientBuilder
	// Resume enable the functionality of resume.
	Resume(opts ...ClientResumeOptions) ClientBuilder
	// Lease enable the functionality of lease.
	Lease() ClientBuilder
	// DataMimeType is used to set payload data MIME type.
	// Default MIME type is `application/binary`.
	DataMimeType(mime string) ClientBuilder
	// MetadataMimeType is used to set payload metadata MIME type.
	// Default MIME type is `application/binary`.
	MetadataMimeType(mime string) ClientBuilder
	// SetupPayload set the setup payload.
	SetupPayload(setup payload.Payload) ClientBuilder
	// ConnectTimeout set connect timeout.
	ConnectTimeout(timeout time.Duration) ClientBuilder
	// OnClose register handler when client socket closed.
	OnClose(func(error)) ClientBuilder
	// OnConnect register handler when client socket connected.
	OnConnect(func(Client, error)) ClientBuilder
	// Acceptor set acceptor for RSocket client.
	Acceptor(acceptor ClientSocketAcceptor) ToClientStarter
}

// ToClientStarter is used to build a RSocket client with custom Transport.
type ToClientStarter interface {
	// Transport set generator func for current RSocket client.
	//
	// Examples:
	//
	// rsocket.TCPClient().SetHostAndPort("127.0.0.1", 7878).Build()
	// rsocket.WebsocketClient().SetURL("ws://127.0.0.1:8080/hello").Build()
	// rsocket.UnixClient().SetPath("/var/run/rsocket.sock").Build()
	Transport(transport.ClientTransporter) ClientStarter
}

type setupClientSocket interface {
	Client
	Setup(ctx context.Context, connectTimeout time.Duration, setup *socket.SetupInfo) error
}

var (
	_ ClientBuilder   = (*clientBuilder)(nil)
	_ ToClientStarter = (*clientBuilder)(nil)
)

type clientBuilder struct {
	reqSche, resSche scheduler.Scheduler
	resume           *resumeOpts
	fragment         int
	tpGen            transport.ClientTransporter
	setup            *socket.SetupInfo
	acceptor         ClientSocketAcceptor
	onCloses         []func(error)
	onConnects       []func(Client, error)
	connectTimeout   time.Duration
}

func (cb *clientBuilder) Scheduler(req, res scheduler.Scheduler) ClientBuilder {
	cb.reqSche = req
	cb.resSche = res
	return cb
}

func (cb *clientBuilder) Lease() ClientBuilder {
	cb.setup.Lease = true
	return cb
}

func (cb *clientBuilder) Resume(opts ...ClientResumeOptions) ClientBuilder {
	if cb.resume == nil {
		cb.resume = newResumeOpts()
	}
	for _, it := range opts {
		it(cb.resume)
	}
	return cb
}

func (cb *clientBuilder) Fragment(mtu int) ClientBuilder {
	if mtu == 0 {
		cb.fragment = fragmentation.MaxFragment
	} else {
		cb.fragment = mtu
	}
	return cb
}

func (cb *clientBuilder) OnConnect(fn func(Client, error)) ClientBuilder {
	cb.onConnects = append(cb.onConnects, fn)
	return cb
}

func (cb *clientBuilder) OnClose(fn func(error)) ClientBuilder {
	cb.onCloses = append(cb.onCloses, fn)
	return cb
}

func (cb *clientBuilder) KeepAlive(tickPeriod, ackTimeout time.Duration, missedAcks int) ClientBuilder {
	cb.setup.KeepaliveInterval = tickPeriod
	cb.setup.KeepaliveLifetime = time.Duration(missedAcks) * ackTimeout
	return cb
}

func (cb *clientBuilder) DataMimeType(mime string) ClientBuilder {
	cb.setup.DataMimeType = bytesconv.StringToBytes(mime)
	return cb
}

func (cb *clientBuilder) MetadataMimeType(mime string) ClientBuilder {
	cb.setup.MetadataMimeType = bytesconv.StringToBytes(mime)
	return cb
}

func (cb *clientBuilder) SetupPayload(setup payload.Payload) ClientBuilder {
	cb.setup.Data = nil
	cb.setup.Metadata = nil

	if data := setup.Data(); len(data) > 0 {
		cb.setup.Data = make([]byte, len(data))
		copy(cb.setup.Data, data)
	}
	if metadata, ok := setup.Metadata(); ok {
		cb.setup.Metadata = make([]byte, len(metadata))
		copy(cb.setup.Metadata, metadata)
	}
	return cb
}

func (cb *clientBuilder) ConnectTimeout(timeout time.Duration) ClientBuilder {
	cb.connectTimeout = timeout
	return cb
}

func (cb *clientBuilder) Acceptor(acceptor ClientSocketAcceptor) ToClientStarter {
	cb.acceptor = acceptor
	return cb
}

func (cb *clientBuilder) Transport(t transport.ClientTransporter) ClientStarter {
	cb.tpGen = t
	return cb
}

func (cb *clientBuilder) Start(ctx context.Context) (client Client, err error) {
	if ctx == nil {
		panic("context cannot be nil!")
	}
	// create a blank socket.
	err = fragmentation.IsValidFragment(cb.fragment)
	if err != nil {
		return
	}

	conn := socket.NewClientDuplexConnection(
		ctx,
		cb.reqSche,
		cb.resSche,
		cb.fragment,
		cb.setup.KeepaliveInterval,
	)
	// create a client.
	var cs setupClientSocket
	if cb.resume != nil {
		cb.setup.Token = cb.resume.tokenGen()
		cs = socket.NewResumableClientSocket(cb.tpGen, conn)
	} else {
		cs = socket.NewClient(cb.tpGen, conn)
	}
	if cb.acceptor != nil {
		conn.SetResponder(cb.acceptor(ctx, cs))
	} else {
		conn.SetResponder(_noopSocket)
	}

	// bind closers.
	if len(cb.onCloses) > 0 {
		for _, closer := range cb.onCloses {
			cs.OnClose(closer)
		}
	}

	// setup client.
	err = cs.Setup(ctx, cb.connectTimeout, cb.setup)

	// destroy connection when setup failed
	if err != nil {
		return
	}

	client = cs

	// trigger OnConnect
	if len(cb.onConnects) > 0 {
		var onConnects []func(Client, error)
		onConnects, cb.onConnects = cb.onConnects, nil
		go func() {
			for _, onConnect := range onConnects {
				onConnect(client, err)
			}
		}()
	}

	return
}

type resumeOpts struct {
	tokenGen func() []byte
}

func newResumeOpts() *resumeOpts {
	return &resumeOpts{
		tokenGen: getPresetResumeTokenGen,
	}
}

func getPresetResumeTokenGen() (token []byte) {
	token, _ = uuid.New().MarshalBinary()
	return
}

// WithClientResumeToken creates a resume token generator.
func WithClientResumeToken(gen func() []byte) ClientResumeOptions {
	return func(opts *resumeOpts) {
		opts.tokenGen = gen
	}
}

// Connect create a new RSocket client builder with default settings.
func Connect() ClientBuilder {
	return &clientBuilder{
		fragment: fragmentation.MaxFragment,
		setup: &socket.SetupInfo{
			Version:           core.DefaultVersion,
			KeepaliveInterval: common.DefaultKeepaliveInterval,
			KeepaliveLifetime: common.DefaultKeepaliveMaxLifetime,
			DataMimeType:      _defaultMimeType,
			MetadataMimeType:  _defaultMimeType,
		},
	}
}
