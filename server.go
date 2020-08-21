package rsocket

import (
	"context"
	"time"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/core/transport"
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
	errUnavailableResume    = []byte("resume not supported")
	errUnavailableLease     = []byte("lease not supported")
	errDuplicatedSetupToken = []byte("duplicated setup token")
)

type (
	// OpServerResume represents resume options for RSocket server.
	OpServerResume func(o *serverResumeOptions)
	// ServerBuilder can be used to build a RSocket server.
	ServerBuilder interface {
		// Fragment set fragmentation size which default is 16_777_215(16MB).
		Fragment(mtu int) ServerBuilder
		// Lease enable feature of Lease.
		Lease(leases lease.Leases) ServerBuilder
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

type serverResumeOptions struct {
	enable          bool
	sessionDuration time.Duration
}

type server struct {
	tp         transport.ServerTransporter
	resumeOpts *serverResumeOptions
	fragment   int
	acc        ServerAcceptor
	sm         *session.Manager
	done       chan struct{}
	onServe    []func()
	leases     lease.Leases
}

func (p *server) Lease(leases lease.Leases) ServerBuilder {
	p.leases = leases
	return p
}

func (p *server) OnStart(onStart func()) ServerBuilder {
	if onStart != nil {
		p.onServe = append(p.onServe, onStart)
	}
	return p
}

func (p *server) Resume(opts ...OpServerResume) ServerBuilder {
	p.resumeOpts.enable = true
	for _, it := range opts {
		it(p.resumeOpts)
	}
	return p
}

func (p *server) Fragment(mtu int) ServerBuilder {
	if mtu == 0 {
		p.fragment = fragmentation.MaxFragment
	} else {
		p.fragment = mtu
	}
	return p
}

func (p *server) Acceptor(acceptor ServerAcceptor) ToServerStarter {
	p.acc = acceptor
	return p
}

func (p *server) Transport(t transport.ServerTransporter) Start {
	p.tp = t
	return p
}

func (p *server) Serve(ctx context.Context) error {
	err := fragmentation.IsValidFragment(p.fragment)
	if err != nil {
		return err
	}

	t, err := p.tp(ctx)
	if err != nil {
		return err
	}

	defer func() {
		_ = t.Close()
	}()

	go func(ctx context.Context) {
		_ = p.loopCleanSession(ctx)
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
				deadline := time.Now().Add(p.resumeOpts.sessionDuration)
				s := session.NewSession(deadline, ssk)
				p.sm.Push(s)
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

		switch frame := first.(type) {
		case *framing.ResumeFrame:
			p.doResume(frame, tp, socketChan)
		case *framing.SetupFrame:
			sendingSocket, err := p.doSetup(frame, tp, socketChan)
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
			err := framing.NewWriteableErrorFrame(0, core.ErrorCodeConnectionError, []byte("first frame must be setup or resume"))
			_ = tp.Send(err, true)
			_ = tp.Close()
			return
		}
		if err := tp.Start(ctx); err != nil {
			logger.Warnf("transport exit: %+v\n", err)
		}
	})

	serveNotifier := make(chan struct{})
	go func(c <-chan struct{}, fn []func()) {
		<-c
		for i := range fn {
			fn[i]()
		}
	}(serveNotifier, p.onServe)
	return t.Listen(ctx, serveNotifier)
}

func (p *server) doSetup(frame *framing.SetupFrame, tp *transport.Transport, socketChan chan<- socket.ServerSocket) (sendingSocket socket.ServerSocket, err *framing.WriteableErrorFrame) {

	if frame.Header().Flag().Check(core.FlagLease) && p.leases == nil {
		err = framing.NewWriteableErrorFrame(0, core.ErrorCodeUnsupportedSetup, errUnavailableLease)
		return
	}

	isResume := frame.Header().Flag().Check(core.FlagResume)

	// 1. receive a token but server doesn't support resume.
	if isResume && !p.resumeOpts.enable {
		err = framing.NewWriteableErrorFrame(0, core.ErrorCodeUnsupportedSetup, errUnavailableResume)
		return
	}

	rawSocket := socket.NewServerDuplexConnection(p.fragment, p.leases)

	// 2. no resume
	if !isResume {
		sendingSocket = socket.NewSimpleServerSocket(rawSocket)
		if responder, e := p.acc(frame, sendingSocket); e != nil {
			err = framing.NewWriteableErrorFrame(0, core.ErrorCodeRejectedSetup, []byte(e.Error()))
		} else {
			sendingSocket.SetResponder(responder)
			sendingSocket.SetTransport(tp)
			socketChan <- sendingSocket
		}
		return
	}

	token := make([]byte, len(frame.Token()))

	// 3. resume reject because of duplicated token.
	if _, ok := p.sm.Load(token); ok {
		err = framing.NewWriteableErrorFrame(0, core.ErrorCodeRejectedSetup, errDuplicatedSetupToken)
		return
	}

	// 4. resume success
	copy(token, frame.Token())
	sendingSocket = socket.NewResumableServerSocket(rawSocket, token)
	if responder, e := p.acc(frame, sendingSocket); e != nil {
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

func (p *server) doResume(frame *framing.ResumeFrame, tp *transport.Transport, socketChan chan<- socket.ServerSocket) {
	var sending core.WriteableFrame
	if !p.resumeOpts.enable {
		sending = framing.NewWriteableErrorFrame(0, core.ErrorCodeRejectedResume, errUnavailableResume)
	} else if s, ok := p.sm.Load(frame.Token()); ok {
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

func (p *server) loopCleanSession(ctx context.Context) (err error) {
	tk := time.NewTicker(serverSessionCleanInterval)
	defer func() {
		tk.Stop()
		p.destroySessions()
	}()
L:
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break L
		case <-p.done:
			break L
		case <-tk.C:
			p.doCleanSession()
		}
	}
	return
}

func (p *server) destroySessions() {
	for p.sm.Len() > 0 {
		nextSession := p.sm.Pop()
		if err := nextSession.Close(); err != nil {
			logger.Warnf("kill session failed: %s\n", err)
		} else if logger.IsDebugEnabled() {
			logger.Debugf("kill session success: %s\n", nextSession)
		}
	}
}

func (p *server) doCleanSession() {
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
	for p.sm.Len() > 0 {
		cur = p.sm.Pop()
		// Push back if session is still alive.
		if !cur.IsDead() {
			p.sm.Push(cur)
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
