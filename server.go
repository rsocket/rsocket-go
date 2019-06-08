package rsocket

import (
	"context"
	"fmt"
	"time"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/fragmentation"
	"github.com/rsocket/rsocket-go/internal/framing"
	"github.com/rsocket/rsocket-go/internal/logger"
	"github.com/rsocket/rsocket-go/internal/session"
	"github.com/rsocket/rsocket-go/internal/socket"
	"github.com/rsocket/rsocket-go/internal/transport"
	"github.com/rsocket/rsocket-go/rx"
)

const (
	serverWorkerPoolSize       = 10000
	serverSessionCleanInterval = 1 * time.Second
	serverSessionDuration      = 30 * time.Second
)

var resumeNotSupportBytes = []byte("resume not supported")

type (
	OpServerResume func(o *serverResumeOptions)
	// ServerBuilder can be used to build a RSocket server.
	ServerBuilder interface {
		// Fragment set fragmentation size which default is 16_777_215(16MB).
		Fragment(mtu int) ServerBuilder
		// Resume enable resume for current server.
		Resume(opts ...OpServerResume) ServerBuilder
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
	return &server{
		fragment:  fragmentation.MaxFragment,
		scheduler: rx.NewElasticScheduler(serverWorkerPoolSize),
		sm:        session.NewManager(),
		done:      make(chan struct{}),
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
	resumeOpts *serverResumeOptions
	fragment   int
	addr       string
	acc        ServerAcceptor
	scheduler  rx.Scheduler
	sm         *session.Manager
	done       chan struct{}
}

func (p *server) Resume(opts ...OpServerResume) ServerBuilder {
	p.resumeOpts.enable = true
	for _, it := range opts {
		it(p.resumeOpts)
	}
	return p
}

func (p *server) Fragment(mtu int) ServerBuilder {
	p.fragment = mtu
	return p
}

func (p *server) Acceptor(acceptor ServerAcceptor) ServerTransportBuilder {
	p.acc = acceptor
	return p
}

func (p *server) Transport(transport string) Start {
	p.addr = transport
	return p
}

func (p *server) Serve() error {
	defer func() {
		_ = p.scheduler.Close()
	}()
	u, err := transport.ParseURI(p.addr)
	if err != nil {
		return err
	}
	err = fragmentation.IsValidFragment(p.fragment)
	if err != nil {
		return err
	}
	t, err := u.MakeServerTransport()
	if err != nil {
		return err
	}

	go func(ctx context.Context) {
		_ = p.loopCleanSession(ctx)
	}(context.Background())

	t.Accept(func(ctx context.Context, tp *transport.Transport) {
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

		tp.HandleResume(func(frame framing.Frame) (err error) {
			defer frame.Release()
			var sending framing.Frame
			if !p.resumeOpts.enable {
				sending = framing.NewFrameError(0, common.ErrorCodeRejectedResume, resumeNotSupportBytes)
			} else if s, ok := p.sm.Load(frame.(*framing.FrameResume).Token()); ok {
				sending = framing.NewResumeOK(0)
				s.Socket().SetTransport(tp)
				socketChan <- s.Socket()
				if logger.IsDebugEnabled() {
					logger.Debugf("recover session: %s\n", s)
				}
			} else {
				sending = framing.NewFrameError(
					0,
					common.ErrorCodeRejectedResume,
					common.Str2bytes(fmt.Sprintf("no such session")),
				)
			}
			if err := tp.Send(sending); err != nil {
				logger.Errorf("send resume response failed: %s\n", err)
				_ = tp.Close()
			}
			sending.Release()
			return
		})
		tp.HandleSetup(func(frame framing.Frame) (err error) {
			defer frame.Release()
			setup := frame.(*framing.FrameSetup)

			isResume := frame.Header().Flag().Check(framing.FlagResume)

			// 1. receive a token but server doesn't support resume.
			if isResume && !p.resumeOpts.enable {
				e := framing.NewFrameError(0, common.ErrorCodeUnsupportedSetup, resumeNotSupportBytes)
				err = tp.Send(e)
				e.Release()
				_ = tp.Close()
				return
			}

			rawSocket := socket.NewServerDuplexRSocket(p.fragment, p.scheduler)
			if !isResume {
				// 2. no resume
				sendingSocket := socket.NewServer(rawSocket)
				sendingSocket.SetResponder(p.acc(setup, sendingSocket))
				sendingSocket.SetTransport(tp)
				go func(ctx context.Context) {
					_ = sendingSocket.Start(ctx)
				}(ctx)
				socketChan <- sendingSocket
			} else {
				// 3. resume
				token := make([]byte, len(setup.Token()))
				copy(token, setup.Token())
				// TODO: process resume
				sendingSocket := socket.NewServerResume(rawSocket, token)
				responder := p.acc(setup, sendingSocket)
				sendingSocket.SetResponder(responder)
				sendingSocket.SetTransport(tp)
				go func(ctx context.Context) {
					_ = sendingSocket.Start(ctx)
				}(ctx)
				socketChan <- sendingSocket
			}
			return
		})

		if err := tp.Start(ctx); err != nil {
			logger.Debugf("transport exit: %s\n", err.Error())
		}
	})
	return t.Listen()
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
		session := p.sm.Pop()
		if err := session.Close(); err != nil {
			logger.Warnf("kill session failed: %s\n", err)
		} else if logger.IsDebugEnabled() {
			logger.Debugf("kill session success: %s\n", session)
		}
	}
}

func (p *server) doCleanSession() {
	deads := make(chan *session.Session)
	go func(deads chan *session.Session) {
		for it := range deads {
			if err := it.Close(); err != nil {
				logger.Warnf("close dead session failed: %s\n", err)
			} else if logger.IsDebugEnabled() {
				logger.Debugf("close dead session success: %s\n", it)
			}
		}
	}(deads)
	var cur *session.Session
	for p.sm.Len() > 0 {
		cur = p.sm.Pop()
		// Push back if session is still alive.
		if !cur.IsDead() {
			p.sm.Push(cur)
			break
		}
		deads <- cur
	}
	close(deads)
}

func WithServerResumeSessionDuration(duration time.Duration) OpServerResume {
	return func(o *serverResumeOptions) {
		o.sessionDuration = duration
	}
}
