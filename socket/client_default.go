package socket

import (
	"context"

	"github.com/rsocket/rsocket-go/internal/logger"
	"github.com/rsocket/rsocket-go/internal/transport"
)

type DefaultClientSocket struct {
	*baseSocket
	uri *transport.URI
}

func (p *DefaultClientSocket) Setup(ctx context.Context, setup *SetupInfo) (err error) {
	tp, err := p.uri.MakeClientTransport()
	if err != nil {
		return
	}
	tp.Connection().SetCounter(p.socket.counter)
	tp.SetLifetime(setup.KeepaliveLifetime)

	p.socket.SetTransport(tp)

	go func(ctx context.Context, tp *transport.Transport) {
		if err := tp.Start(ctx); err != nil {
			logger.Debugf("client exit: %s\n", err)
		}
		_ = p.Close()
	}(ctx, tp)

	go func(ctx context.Context) {
		_ = p.socket.loopWrite(ctx)
	}(ctx)
	setupFrame := setup.ToFrame()
	err = p.socket.tp.Send(setupFrame)
	setupFrame.Release()
	return
}

func NewClient(uri *transport.URI, socket *DuplexRSocket) *DefaultClientSocket {
	return &DefaultClientSocket{
		baseSocket: newBaseSocket(socket),
		uri:        uri,
	}
}
