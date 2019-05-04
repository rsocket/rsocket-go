package rsocket

import (
	"context"
	"fmt"
	"time"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/fragmentation"
	"github.com/rsocket/rsocket-go/internal/framing"
	"github.com/rsocket/rsocket-go/internal/logger"
	"github.com/rsocket/rsocket-go/internal/transport"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/socket"
)

const (
	serverWorkerPoolSize       = 10000
	serverSessionCleanInterval = 1 * time.Second
	serverSessionDuration      = 30 * time.Second
)

var resumeNotSupportBytes = []byte("resume not supported")

type (
	// ServerBuilder can be used to build a RSocket server.
	ServerBuilder interface {
		// Fragment set fragmentation size which default is 16_777_215(16MB).
		Fragment(mtu int) ServerBuilder
		// Resume enable resume for current server.
		Resume() ServerBuilder
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
		fragment:              fragmentation.MaxFragment,
		scheduler:             rx.NewElasticScheduler(serverWorkerPoolSize),
		sm:                    NewSessionManager(),
		done:                  make(chan struct{}),
		resumeSessionDuration: serverSessionDuration,
	}
}

type server struct {
	resumeEnable          bool
	resumeSessionDuration time.Duration

	fragment  int
	addr      string
	acc       ServerAcceptor
	scheduler rx.Scheduler
	sm        *SessionManager
	done      chan struct{}
}

func (p *server) Resume() ServerBuilder {
	p.resumeEnable = true
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
			case s, ok := <-socketChan:
				if !ok {
					break
				}
				_, ok = s.Token()
				if !ok {
					_ = s.Close()
					break
				}
				s.Pause()
				deadline := time.Now().Add(p.resumeSessionDuration)
				session := NewSession(deadline, s)
				p.sm.Push(session)
				if logger.IsDebugEnabled() {
					logger.Debugf("store session: %s\n", session)
				}
			default:
			}
			close(socketChan)
		}()

		tp.HandleResume(func(frame framing.Frame) (err error) {
			defer frame.Release()
			var sending framing.Frame
			if !p.resumeEnable {
				sending = framing.NewFrameError(0, common.ErrorCodeRejectedResume, resumeNotSupportBytes)
			} else if session, ok := p.sm.Load(frame.(*framing.FrameResume).Token()); ok {
				sending = framing.NewResumeOK(0)
				session.socket.SetTransport(tp)
				socketChan <- session.socket
				if logger.IsDebugEnabled() {
					logger.Debugf("recover session: %s\n", session)
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
			if isResume && !p.resumeEnable {
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
	defer tk.Stop()
L:
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break L
		case <-p.done:
			break L
		case <-tk.C:
			p.bingoCleanSession()
		}
	}
	return
}

func (p *server) bingoCleanSession() {
	deads := make(chan *Session)
	go func(deads chan *Session) {
		for it := range deads {
			_ = it.Close()
			if logger.IsDebugEnabled() {
				logger.Debugf("close dead session: %s\n", it)
			}
		}
	}(deads)
	var cur *Session
	for p.sm.Len() > 0 {
		cur = p.sm.Pop()
		if !cur.IsDead() {
			p.sm.Push(cur)
			break
		}
		deads <- cur
	}
	close(deads)
}
