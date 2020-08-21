package socket

import (
	"context"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/logger"
)

type simpleClientSocket struct {
	*BaseSocket
	tp transport.ClientTransporter
}

func (p *simpleClientSocket) Setup(ctx context.Context, setup *SetupInfo) (err error) {
	tp, err := p.tp(ctx)
	if err != nil {
		return
	}
	tp.Connection().SetCounter(p.socket.counter)
	tp.SetLifetime(setup.KeepaliveLifetime)

	p.socket.SetTransport(tp)

	if setup.Lease {
		p.refreshLease(0, 0)
		tp.RegisterHandler(transport.OnLease, func(frame core.Frame) (err error) {
			lease := frame.(*framing.LeaseFrame)
			p.refreshLease(lease.TimeToLive(), int64(lease.NumberOfRequests()))
			return
		})
	}

	tp.RegisterHandler(transport.OnErrorWithZeroStreamID, func(frame core.Frame) (err error) {
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
	return
}

// NewClient create a simple client-side socket.
func NewClient(tp transport.ClientTransporter, socket *DuplexConnection) ClientSocket {
	return &simpleClientSocket{
		BaseSocket: NewBaseSocket(socket),
		tp:         tp,
	}
}
