package socket

import (
	"context"
	"time"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/logger"
)

type simpleClientSocket struct {
	*BaseSocket
	tp transport.ClientTransporter
}

func (p *simpleClientSocket) Setup(ctx context.Context, connectTimeout time.Duration, setup *SetupInfo) (err error) {
	tp, err := p.createTransport(ctx, connectTimeout)
	if err != nil {
		return
	}
	tp.Connection().SetCounter(p.socket.counter)
	tp.SetLifetime(setup.KeepaliveLifetime)

	p.socket.SetTransport(tp)

	if setup.Lease {
		p.refreshLease(0, 0)
		tp.Handle(transport.OnLease, func(frame core.BufferedFrame) (err error) {
			lease := frame.(*framing.LeaseFrame)
			p.refreshLease(lease.TimeToLive(), int64(lease.NumberOfRequests()))
			return
		})
	}

	tp.Handle(transport.OnErrorWithZeroStreamID, func(frame core.BufferedFrame) (err error) {
		p.socket.SetError(frame.(*framing.ErrorFrame))
		return
	})

	go func(ctx context.Context, tp *transport.Transport) {
		if err := tp.Start(ctx); err != nil {
			logger.Warnf("client exit failed: %+v\n", err)
		}
		_ = p.Close()
	}(ctx, tp)

	go func() {
		_ = p.socket.LoopWrite(ctx)
	}()
	setupFrame := setup.toFrame()
	err = p.socket.tp.Send(setupFrame, true)
	if err != nil {
		_ = p.close(false)
	}
	return
}

func (p *simpleClientSocket) createTransport(ctx context.Context, connectTimeout time.Duration) (*transport.Transport, error) {
	var tpCtx = ctx
	if connectTimeout > 0 {
		c, cancel := context.WithTimeout(ctx, connectTimeout)
		tpCtx = c
		defer cancel()
	}
	return p.tp(tpCtx)
}

// NewClient create a simple client-side socket.
func NewClient(tp transport.ClientTransporter, socket *DuplexConnection) ClientSocket {
	return &simpleClientSocket{
		BaseSocket: NewBaseSocket(socket),
		tp:         tp,
	}
}
