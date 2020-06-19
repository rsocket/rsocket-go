package socket

import (
	"context"
	"crypto/tls"

	"github.com/rsocket/rsocket-go/internal/framing"
	"github.com/rsocket/rsocket-go/internal/transport"
	"github.com/rsocket/rsocket-go/logger"
)

type defaultClientSocket struct {
	*baseSocket
	uri     *transport.URI
	headers map[string][]string
	tls     *tls.Config
}

func (p *defaultClientSocket) Setup(ctx context.Context, setup *SetupInfo) (err error) {
	tp, err := p.uri.MakeClientTransport(p.tls, p.headers)
	if err != nil {
		return
	}
	tp.Connection().SetCounter(p.socket.counter)
	tp.SetLifetime(setup.KeepaliveLifetime)

	p.socket.SetTransport(tp)

	if setup.Lease {
		p.refreshLease(0, 0)
		tp.HandleLease(func(frame framing.Frame) (err error) {
			lease := frame.(*framing.LeaseFrame)
			p.refreshLease(lease.TimeToLive(), int64(lease.NumberOfRequests()))
			logger.Infof(">>>>> refresh lease: %v\n", lease)
			return
		})
	}

	tp.HandleDisaster(func(frame framing.Frame) (err error) {
		p.socket.SetError(frame.(*framing.ErrorFrame))
		return
	})

	go func(ctx context.Context, tp *transport.Transport) {
		if err := tp.Start(ctx); err != nil {
			logger.Warnf("client exit failed: %+v\n", err)
		}
		_ = p.Close()
	}(ctx, tp)

	go func(ctx context.Context) {
		_ = p.socket.loopWrite(ctx)
	}(ctx)
	setupFrame := setup.toFrame()
	err = p.socket.tp.Send(setupFrame, true)
	return
}

// NewClient create a simple client-side socket.
func NewClient(uri *transport.URI, socket *DuplexRSocket, tc *tls.Config, headers map[string][]string) ClientSocket {
	return &defaultClientSocket{
		baseSocket: newBaseSocket(socket),
		uri:        uri,
		headers:    headers,
		tls:        tc,
	}
}
