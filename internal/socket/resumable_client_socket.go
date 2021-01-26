package socket

import (
	"context"
	"math"
	"time"

	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/logger"
	"go.uber.org/atomic"
)

const _resumeReconnectDelay = 1 * time.Second
const _resumeTimeout = 10 * time.Second

type resumeClientSocket struct {
	*BaseSocket
	connects *atomic.Int32
	setup    *SetupInfo
	tp       transport.ClientTransporter
}

func (r *resumeClientSocket) Setup(ctx context.Context, timeout time.Duration, setup *SetupInfo) error {
	r.setup = setup
	go func(ctx context.Context) {
		_ = r.socket.LoopWrite(ctx)
	}(ctx)
	return r.connect(ctx, timeout)
}

func (r *resumeClientSocket) Close() (err error) {
	r.once.Do(func() {
		r.markAsClosing()
		err = r.socket.Close()
		for i, l := 0, len(r.closers); i < l; i++ {
			r.closers[l-i-1](err)
		}
	})
	return
}

func (r *resumeClientSocket) createTransport(ctx context.Context, timeout time.Duration) (*transport.Transport, error) {
	var tpCtx = ctx
	if timeout > 0 {
		c, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		tpCtx = c
	}
	return r.tp(tpCtx)
}

func (r *resumeClientSocket) connect(ctx context.Context, timeout time.Duration) (err error) {
	connects := r.connects.Inc()
	if connects < 0 {
		_ = r.Close()
		return
	}
	tp, err := r.createTransport(ctx, timeout)
	if err != nil {
		if connects == 1 {
			return
		}
		time.Sleep(_resumeReconnectDelay)
		_ = r.connect(ctx, timeout)
		return
	}
	tp.Connection().SetCounter(r.socket.counter)
	tp.SetLifetime(r.setup.KeepaliveLifetime)

	go func(ctx context.Context, tp *transport.Transport) {
		defer func() {
			r.socket.clearTransport()
			if r.isClosed() {
				_ = r.Close()
				return
			}
			time.Sleep(_resumeReconnectDelay)
			_ = r.connect(ctx, timeout)
		}()
		err := tp.Start(ctx)
		if err != nil && logger.IsDebugEnabled() {
			logger.Debugf("resumable client stopped: %s\n", err)
		}
	}(ctx, tp)

	// connect first time.
	if len(r.setup.Token) < 1 || connects == 1 {
		tp.Handle(transport.OnErrorWithZeroStreamID, func(frame core.BufferedFrame) (err error) {
			defer frame.Release()
			r.socket.SetError(frame.(*framing.ErrorFrame).ToError())
			r.markAsClosing()
			return
		})
		err = tp.Send(r.setup.toFrame(), true)
		r.socket.SetTransport(tp)
		// destroy if setup failed
		if err != nil {
			_ = r.close(false)
		}
		return
	}

	resumeErr := make(chan error)

	tp.Handle(transport.OnResumeOK, func(frame core.BufferedFrame) (err error) {
		defer frame.Release()
		close(resumeErr)
		return
	})

	tp.Handle(transport.OnErrorWithZeroStreamID, func(frame core.BufferedFrame) error {
		defer frame.Release()
		// TODO: process other error with zero StreamID
		errFrame := frame.(*framing.ErrorFrame)
		if errFrame.ErrorCode() == core.ErrorCodeRejectedResume {
			defer func() {
				if err := recover(); err != nil {
					logger.Warnf("handle reject resume failed: %s\n", err)
				}
			}()
			resumeErr <- errFrame.ToError()
			close(resumeErr)
		}
		return nil
	})

	err = tp.Send(framing.NewWriteableResumeFrame(
		core.DefaultVersion,
		r.setup.Token,
		r.socket.counter.WriteBytes(),
		r.socket.counter.ReadBytes(),
	), true)

	if err != nil {
		// destroy if resume failed
		_ = r.close(false)
		return err
	}

	select {
	case <-time.After(_resumeTimeout):
		err = errors.New("resume timeout")
		// destroy if resume failed
		_ = r.close(false)
	case reject, ok := <-resumeErr:
		if ok {
			logger.Errorf("resume failed: %s\n", reject.Error())
			r.markAsClosing()
			err = r.connect(ctx, timeout)
		} else {
			r.socket.SetTransport(tp)
		}
	}
	return
}

func (r *resumeClientSocket) markAsClosing() {
	r.connects.Store(math.MinInt32)
}

func (r *resumeClientSocket) isClosed() bool {
	return r.connects.Load() < 0
}

// NewResumableClientSocket creates a client-side socket with resume support.
func NewResumableClientSocket(tp transport.ClientTransporter, socket *DuplexConnection) ClientSocket {
	return &resumeClientSocket{
		BaseSocket: NewBaseSocket(socket),
		connects:   atomic.NewInt32(0),
		tp:         tp,
	}
}
