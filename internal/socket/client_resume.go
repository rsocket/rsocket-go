package socket

import (
	"context"
	"errors"
	"math"
	"sync/atomic"
	"time"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/framing"
	"github.com/rsocket/rsocket-go/internal/logger"
	"github.com/rsocket/rsocket-go/internal/transport"
)

const reconnectDelay = 1 * time.Second

type resumeClientSocket struct {
	*baseSocket
	connects int32
	uri      *transport.URI
	setup    *SetupInfo
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
			p.closers[l-i-1]()
		}
	})
	return
}

func (p *resumeClientSocket) connect(ctx context.Context) (err error) {
	connects := atomic.AddInt32(&(p.connects), 1)
	if connects < 0 {
		_ = p.Close()
		return
	}
	tp, err := p.uri.MakeClientTransport()
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
		tp.HandleError0(func(frame framing.Frame) (err error) {
			p.markClosing()
			frame.Release()
			return
		})

		f = p.setup.ToFrame()
		err = tp.Send(f)
		f.Release()
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
		frame.Release()
		return
	})

	tp.HandleError0(func(frame framing.Frame) (err error) {
		// TODO: process other error with zero StreamID
		f := frame.(*framing.FrameError)
		if f.ErrorCode() == common.ErrorCodeRejectedResume {
			resumeErr <- f.Error()
			close(resumeErr)
		}
		frame.Release()
		return
	})

	err = tp.Send(f)
	f.Release()
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
	atomic.StoreInt32(&(p.connects), math.MinInt32)
}

func (p *resumeClientSocket) isClosed() bool {
	return atomic.LoadInt32(&(p.connects)) < 0
}

// NewClientResume creates a client-side socket with resume support.
func NewClientResume(uri *transport.URI, socket *DuplexRSocket) ClientSocket {
	return &resumeClientSocket{
		baseSocket: newBaseSocket(socket),
		uri:        uri,
	}
}
