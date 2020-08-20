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

const _resumeReconnectDelay = 1 * time.Second
const _resumeTimeout = 10 * time.Second

type resumeClientSocket struct {
	*BaseSocket
	connects *atomic.Int32
	setup    *SetupInfo
	tp       transport.ClientTransportFunc
}

func (r *resumeClientSocket) Setup(ctx context.Context, setup *SetupInfo) error {
	r.setup = setup
	go func(ctx context.Context) {
		_ = r.socket.LoopWrite(ctx)
	}(ctx)
	return r.connect(ctx)
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

func (r *resumeClientSocket) connect(ctx context.Context) (err error) {
	connects := r.connects.Inc()
	if connects < 0 {
		_ = r.Close()
		return
	}
	tp, err := r.tp(ctx)
	if err != nil {
		if connects == 1 {
			return
		}
		time.Sleep(_resumeReconnectDelay)
		_ = r.connect(ctx)
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
			_ = r.connect(ctx)
		}()
		err := tp.Start(ctx)
		if err != nil && logger.IsDebugEnabled() {
			logger.Debugf("resumable client stopped: %s\n", err)
		}
	}(ctx, tp)

	var f core.WriteableFrame

	// connect first time.
	if len(r.setup.Token) < 1 || connects == 1 {
		tp.RegisterHandler(transport.OnErrorWithZeroStreamID, func(frame core.Frame) (err error) {
			r.socket.SetError(frame.(*framing.ErrorFrame))
			r.markAsClosing()
			return
		})
		f = r.setup.toFrame()
		err = tp.Send(f, true)
		r.socket.SetTransport(tp)
		return
	}

	f = framing.NewWriteableResumeFrame(
		core.DefaultVersion,
		r.setup.Token,
		r.socket.counter.WriteBytes(),
		r.socket.counter.ReadBytes(),
	)

	resumeErr := make(chan string)

	tp.RegisterHandler(transport.OnResumeOK, func(frame core.Frame) (err error) {
		close(resumeErr)
		return
	})

	tp.RegisterHandler(transport.OnErrorWithZeroStreamID, func(frame core.Frame) error {
		// TODO: process other error with zero StreamID
		f := frame.(*framing.ErrorFrame)
		if f.ErrorCode() == core.ErrorCodeRejectedResume {
			defer func() {
				if err := recover(); err != nil {
					logger.Warnf("handle reject resume failed: %s\n", err)
				}
			}()
			resumeErr <- f.Error()
			close(resumeErr)
		}
		return nil
	})

	err = tp.Send(f, true)
	if err != nil {
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, _resumeTimeout)
	defer cancel()

	select {
	case <-timeoutCtx.Done():
		err = timeoutCtx.Err()
	case reject, ok := <-resumeErr:
		if ok {
			err = errors.New(reject)
			r.markAsClosing()
			err = r.connect(ctx)
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
func NewResumableClientSocket(tp transport.ClientTransportFunc, socket *DuplexConnection) ClientSocket {
	return &resumeClientSocket{
		BaseSocket: NewBaseSocket(socket),
		connects:   atomic.NewInt32(0),
		tp:         tp,
	}
}
