package socket

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/logger"
	"go.uber.org/atomic"
)

const reconnectDelay = 1 * time.Second

type resumeClientSocket struct {
	*BaseSocket
	connects *atomic.Int32
	setup    *SetupInfo
	tp       transport.ClientTransportFunc
}

func (p *resumeClientSocket) Setup(ctx context.Context, setup *SetupInfo) error {
	p.setup = setup
	go func(ctx context.Context) {
		_ = p.socket.LoopWrite(ctx)
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
	tp, err := p.tp(ctx)
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

	var f core.WriteableFrame

	// connect first time.
	if len(p.setup.Token) < 1 || connects == 1 {
		tp.RegisterHandler(transport.OnErrorWithZeroStreamID, func(frame core.Frame) (err error) {
			p.socket.SetError(frame.(*framing.ErrorFrame))
			p.markClosing()
			return
		})
		f = p.setup.toFrame()
		err = tp.Send(f, true)
		p.socket.SetTransport(tp)
		return
	}

	f = framing.NewWriteableResumeFrame(
		core.DefaultVersion,
		p.setup.Token,
		p.socket.counter.WriteBytes(),
		p.socket.counter.ReadBytes(),
	)

	resumeErr := make(chan string)

	tp.RegisterHandler(transport.OnResumeOK, func(frame core.Frame) (err error) {
		close(resumeErr)
		return
	})

	tp.RegisterHandler(transport.OnErrorWithZeroStreamID, func(frame core.Frame) (err error) {
		// TODO: process other error with zero StreamID
		f := frame.(*framing.ErrorFrame)
		if f.ErrorCode() == core.ErrorCodeRejectedResume {
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

// NewResumableClientSocket creates a client-side socket with resume support.
func NewResumableClientSocket(tp transport.ClientTransportFunc, socket *DuplexConnection) ClientSocket {
	return &resumeClientSocket{
		BaseSocket: NewBaseSocket(socket),
		connects:   atomic.NewInt32(0),
		tp:         tp,
	}
}
