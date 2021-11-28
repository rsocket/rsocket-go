package rsocket

import (
	"context"
	"time"

	"github.com/jjeffcaii/reactor-go/scheduler"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/internal/bytesconv"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/fragmentation"
	"github.com/rsocket/rsocket-go/internal/session"
	"github.com/rsocket/rsocket-go/internal/socket"
	"github.com/rsocket/rsocket-go/lease"
	"github.com/rsocket/rsocket-go/logger"
)

const (
	serverSessionCleanInterval = 500 * time.Millisecond
	serverSessionDuration      = 30 * time.Second
)

var (
	_errUnavailableResume    = "resume not supported"
	_errUnavailableLease     = "lease not supported"
	_errDuplicatedSetupToken = "duplicated setup token"
	_errInvalidFirstFrame    = "first frame must be setup or resume"
)

type (
	// OpServerResume represents resume options for RSocket server.
	OpServerResume func(o *serverResumeOptions)
	// ServerBuilder can be used to build a RSocket server.
	ServerBuilder interface {
		// Scheduler set schedulers for the requests or responses.
		// Nil scheduler means keep the default scheduler settings.
		Scheduler(requestScheduler, responseScheduler scheduler.Scheduler) ServerBuilder
		// Fragment set fragmentation size which default is 16_777_215(16MB).
		Fragment(mtu int) ServerBuilder
		// Lease enable feature of Lease.
		Lease(leases lease.Factory) ServerBuilder
		// Resume enable resume for current server.
		Resume(opts ...OpServerResume) ServerBuilder
		// Acceptor register server acceptor which is used to handle incoming RSockets.
		Acceptor(acceptor ServerAcceptor) ToServerStarter
		// OnStart register a handler when serve success.
		OnStart(onStart func()) ServerBuilder
	}

	// ToServerStarter is used to build a RSocket server with custom Transport string.
	ToServerStarter interface {
		// Transport specify transport generator func.
		// Example:
		// rsocket.TCPServer().SetAddr(":8888").Build()
		Transport(t transport.ServerTransporter) Start
	}

	// Start start a RSocket server.
	Start interface {
		// Serve serve RSocket server.
		Serve(ctx context.Context) error
	}
)

type serverResumeOptions struct {
	enable          bool
	sessionDuration time.Duration
}

var (
	_ ServerBuilder   = (*server)(nil)
	_ ToServerStarter = (*server)(nil)
)

type server struct {
	reqSc, resSc scheduler.Scheduler
	tp           transport.ServerTransporter
	resumeOpts   *serverResumeOptions
	fragment     int
	acc          ServerAcceptor
	sm           *session.Manager
	done         chan struct{}
	onServe      []func()
	leases       lease.Factory
}

func (srv *server) Scheduler(req, res scheduler.Scheduler) ServerBuilder {
	srv.reqSc = req
	srv.resSc = res
	return srv
}

func (srv *server) Lease(leases lease.Factory) ServerBuilder {
	srv.leases = leases
	return srv
}

func (srv *server) OnStart(onStart func()) ServerBuilder {
	if onStart != nil {
		srv.onServe = append(srv.onServe, onStart)
	}
	return srv
}

func (srv *server) Resume(opts ...OpServerResume) ServerBuilder {
	srv.resumeOpts.enable = true
	for _, it := range opts {
		it(srv.resumeOpts)
	}
	return srv
}

func (srv *server) Fragment(mtu int) ServerBuilder {
	if mtu == 0 {
		srv.fragment = fragmentation.MaxFragment
	} else {
		srv.fragment = mtu
	}
	return srv
}

func (srv *server) Acceptor(acceptor ServerAcceptor) ToServerStarter {
	srv.acc = acceptor
	return srv
}

func (srv *server) Transport(t transport.ServerTransporter) Start {
	srv.tp = t
	return srv
}

func (srv *server) Serve(ctx context.Context) error {
	if ctx == nil {
		panic("context cannot be nil!")
	}

	err := fragmentation.IsValidFragment(srv.fragment)
	if err != nil {
		return err
	}

	t, err := srv.tp(ctx)
	if err != nil {
		return err
	}

	defer func() {
		_ = t.Close()
	}()

	go func(ctx context.Context) {
		_ = srv.loopCleanSession(ctx)
	}(ctx)
	t.Accept(func(ctx context.Context, tp *transport.Transport, onClose func(*transport.Transport)) {
		defer onClose(tp)
		socketChan := make(chan socket.ServerSocket, 1)
		defer func() {
			select {
			case ssk, ok := <-socketChan:
				if !ok {
					break
				}
				_, ok = ssk.Token()
				if !ok {
					_ = ssk.Close()
					break
				}
				ssk.Pause()
				deadline := time.Now().Add(srv.resumeOpts.sessionDuration)
				s := session.NewSession(deadline, ssk)
				srv.sm.Push(s)
				if logger.IsDebugEnabled() {
					logger.Debugf("store session: %s\n", s)
				}
			default:
			}
			close(socketChan)
		}()

		first, err := tp.ReadFirst(ctx)
		if err != nil {
			logger.Errorf("read first frame failed: %s\n", err)
			_ = tp.Close()
			return
		}

		defer first.Release()

		switch frame := first.(type) {
		case *framing.ResumeFrame:
			srv.doResume(frame, tp, socketChan)
		case *framing.SetupFrame:
			sendingSocket, err := srv.doSetup(ctx, frame, tp, socketChan)
			if err != nil {
				_ = tp.Send(err, true)
				_ = tp.Close()
				return
			}
			go func(ctx context.Context, sendingSocket socket.ServerSocket) {
				if err := sendingSocket.Start(ctx); err != nil && logger.IsDebugEnabled() {
					logger.Debugf("sending socket exit: %w\n", err)
				}
			}(ctx, sendingSocket)
		default:
			err := framing.NewWriteableErrorFrame(0, core.ErrorCodeConnectionError, bytesconv.StringToBytes(_errInvalidFirstFrame))
			_ = tp.Send(err, true)
			_ = tp.Close()
			return
		}
		if err := tp.Start(ctx); err != nil {
			logger.Warnf("transport exit: %+v\n", err)
		}
	})

	notifier := make(chan bool)
	go func(c <-chan bool, fn []func()) {
		<-c
		for i := range fn {
			fn[i]()
		}
	}(notifier, srv.onServe)
	return t.Listen(ctx, notifier)
}

func (srv *server) doSetup(ctx context.Context, frame *framing.SetupFrame, tp *transport.Transport, socketChan chan<- socket.ServerSocket) (sendingSocket socket.ServerSocket, err *framing.WriteableErrorFrame) {
	if frame.HasFlag(core.FlagLease) && srv.leases == nil {
		err = framing.NewWriteableErrorFrame(0, core.ErrorCodeUnsupportedSetup, bytesconv.StringToBytes(_errUnavailableLease))
		return
	}

	isResume := frame.HasFlag(core.FlagResume)

	// 1. receive a token but server doesn't support resume.
	if isResume && !srv.resumeOpts.enable {
		err = framing.NewWriteableErrorFrame(0, core.ErrorCodeUnsupportedSetup, bytesconv.StringToBytes(_errUnavailableResume))
		return
	}

	rawSocket := socket.NewServerDuplexConnection(ctx, srv.reqSc, srv.resSc, srv.fragment, srv.leases)

	// 2. no resume
	if !isResume {
		sendingSocket = socket.NewSimpleServerSocket(rawSocket)

		// workaround: set address info
		if addr, ok := tp.Addr(); ok {
			sendingSocket.SetAddr(addr)
		}

		if responder, e := srv.acc(ctx, frame, sendingSocket); e != nil {
			err = framing.NewWriteableErrorFrame(0, core.ErrorCodeRejectedSetup, []byte(e.Error()))
		} else {
			sendingSocket.SetResponder(responder)
			sendingSocket.SetTransport(tp)
			socketChan <- sendingSocket
		}
		return
	}

	token := frame.Token()

	// 3. resume reject because of duplicated token.
	if _, ok := srv.sm.Load(token); ok {
		err = framing.NewWriteableErrorFrame(0, core.ErrorCodeRejectedSetup, bytesconv.StringToBytes(_errDuplicatedSetupToken))
		return
	}

	// clone token
	token = common.CloneBytes(token)

	// 4. resume success
	sendingSocket = socket.NewResumableServerSocket(rawSocket, token)

	// workaround: set address info
	if addr, ok := tp.Addr(); ok {
		sendingSocket.SetAddr(addr)
	}

	if responder, e := srv.acc(ctx, frame, sendingSocket); e != nil {
		switch vv := e.(type) {
		case *framing.ErrorFrame:
			err = framing.NewWriteableErrorFrame(0, vv.ErrorCode(), vv.ErrorData())
		default:
			err = framing.NewWriteableErrorFrame(0, core.ErrorCodeInvalidSetup, []byte(e.Error()))
		}
	} else {
		sendingSocket.SetResponder(responder)
		sendingSocket.SetTransport(tp)
		socketChan <- sendingSocket
	}
	return
}

func (srv *server) doResume(frame *framing.ResumeFrame, tp *transport.Transport, socketChan chan<- socket.ServerSocket) {
	var sending core.WriteableFrame
	if !srv.resumeOpts.enable {
		sending = framing.NewWriteableErrorFrame(0, core.ErrorCodeRejectedResume, bytesconv.StringToBytes(_errUnavailableResume))
	} else if s, ok := srv.sm.Load(frame.Token()); ok {
		sending = framing.NewWriteableResumeOKFrame(0)
		s.Socket().SetTransport(tp)
		socketChan <- s.Socket()
		if logger.IsDebugEnabled() {
			logger.Debugf("recover session: %s\n", s)
		}
	} else {
		sending = framing.NewWriteableErrorFrame(
			0,
			core.ErrorCodeRejectedResume,
			[]byte("no such session"),
		)
	}
	if err := tp.Send(sending, true); err != nil {
		logger.Errorf("send resume response failed: %s\n", err)
		_ = tp.Close()
	}
}

func (srv *server) loopCleanSession(ctx context.Context) (err error) {
	tk := time.NewTicker(serverSessionCleanInterval)
	defer func() {
		tk.Stop()
		srv.destroySessions()
	}()
L:
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break L
		case <-srv.done:
			break L
		case <-tk.C:
			srv.doCleanSession()
		}
	}
	return
}

func (srv *server) destroySessions() {
	for srv.sm.Len() > 0 {
		nextSession := srv.sm.Pop()
		if err := nextSession.Close(); err != nil {
			logger.Warnf("kill session failed: %s\n", err)
		} else if logger.IsDebugEnabled() {
			logger.Debugf("kill session success: %s\n", nextSession)
		}
	}
}

func (srv *server) doCleanSession() {
	deadSessions := make(chan *session.Session)
	go func(deadSessions chan *session.Session) {
		for it := range deadSessions {
			if err := it.Close(); err != nil {
				logger.Warnf("close dead session failed: %s\n", err)
			} else if logger.IsDebugEnabled() {
				logger.Debugf("close dead session success: %s\n", it)
			}
		}
	}(deadSessions)
	var cur *session.Session
	for srv.sm.Len() > 0 {
		cur = srv.sm.Pop()
		// Push back if session is still alive.
		if !cur.IsDead() {
			srv.sm.Push(cur)
			break
		}
		deadSessions <- cur
	}
	close(deadSessions)
}

// WithServerResumeSessionDuration sets resume session duration for RSocket server.
func WithServerResumeSessionDuration(duration time.Duration) OpServerResume {
	return func(o *serverResumeOptions) {
		o.sessionDuration = duration
	}
}

// Receive receives server connections from client RSockets.
func Receive() ServerBuilder {
	return &server{
		fragment: fragmentation.MaxFragment,
		sm:       session.NewManager(),
		done:     make(chan struct{}),
		resumeOpts: &serverResumeOptions{
			sessionDuration: serverSessionDuration,
		},
	}
}
