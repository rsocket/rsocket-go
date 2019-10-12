package socket

import (
	"context"
	"crypto/tls"
	"errors"
	"math"
	"time"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/framing"
	"github.com/rsocket/rsocket-go/internal/transport"
	"github.com/rsocket/rsocket-go/logger"
	"go.uber.org/atomic"
)

const reconnectDelay = 1 * time.Second

type resumeClientSocket struct {
	*baseSocket
	connects *atomic.Int32
	uri      *transport.URI
	headers  map[string][]string
	setup    *SetupInfo
	tc       *tls.Config
}

func (p *resumeClientSocket) Setup(ctx context.Context, setup *SetupInfo) error {
	p.setup = setup
	go func(ctx context.Context) {
		_ = p.socket.loopWrite(ctx)
	}(ctx)
	return p.connect(ctx)
}

func (p *resumeClientSocket) Close() (err error) {
	p.once.Do(func() {
		p.markClosing()
		err = p.socket.Close()
		for i, l := 0, len(p.closers); i < l; i++ {
			p.closers[l-i-1](err)
		}
	})
	return
}

func (p *resumeClientSocket) connect(ctx context.Context) (err error) {
	connects := p.connects.Inc()
	if connects < 0 {
		_ = p.Close()
		return
	}
	tp, err := p.uri.MakeClientTransport(p.tc, p.headers)
	if err != nil {
		if connects == 1 {
			return
		}
		time.Sleep(reconnectDelay)
		_ = p.connect(ctx)
		return
	}
	tp.Connection().SetCounter(p.socket.counter)
	tp.SetLifetime(p.setup.KeepaliveLifetime)

	go func(ctx context.Context, tp *transport.Transport) {
		defer func() {
			p.socket.clearTransport()
			if p.isClosed() {
				_ = p.Close()
				return
			}
			time.Sleep(reconnectDelay)
			_ = p.connect(ctx)
		}()
		err := tp.Start(ctx)
		if err != nil {
			logger.Errorf("client exit: %s\n", err)
		}
	}(ctx, tp)

	var f framing.Frame

	// connect first time.
	if len(p.setup.Token) < 1 || connects == 1 {
		tp.HandleDisaster(func(frame framing.Frame) (err error) {
			p.socket.SetError(frame.(*framing.FrameError))
			p.markClosing()
			return
		})

		f = p.setup.toFrame()
		err = tp.Send(f, true)
		p.socket.SetTransport(tp)
		return
	}

	f = framing.NewFrameResume(
		common.DefaultVersion,
		p.setup.Token,
		p.socket.counter.WriteBytes(),
		p.socket.counter.ReadBytes(),
	)

	resumeErr := make(chan string)

	tp.HandleResumeOK(func(frame framing.Frame) (err error) {
		close(resumeErr)
		return
	})

	tp.HandleDisaster(func(frame framing.Frame) (err error) {
		// TODO: process other error with zero StreamID
		f := frame.(*framing.FrameError)
		if f.ErrorCode() == common.ErrorCodeRejectedResume {
			resumeErr <- f.Error()
			close(resumeErr)
		}
		return
	})

	err = tp.Send(f, true)
	if err != nil {
		return err
	}

	ctx2, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	select {
	case <-ctx2.Done():
		err = ctx2.Err()
	case reject, ok := <-resumeErr:
		if ok {
			err = errors.New(reject)
			p.markClosing()
		} else {
			p.socket.SetTransport(tp)
		}
	}
	return
}

func (p *resumeClientSocket) markClosing() {
	p.connects.Store(math.MinInt32)
}

func (p *resumeClientSocket) isClosed() bool {
	return p.connects.Load() < 0
}

// NewClientResume creates a client-side socket with resume support.
func NewClientResume(uri *transport.URI, socket *DuplexRSocket, tc *tls.Config, headers map[string][]string) ClientSocket {
	return &resumeClientSocket{
		baseSocket: newBaseSocket(socket),
		uri:        uri,
		tc:         tc,
		headers:    headers,
		connects:   atomic.NewInt32(0),
	}
}
