package socket

import (
	"context"

	"github.com/jjeffcaii/reactor-go"
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
	"go.uber.org/atomic"
)

// FinalPayload is a marker interface for payloads that should be sent with FlagNext|FlagComplete.
type FinalPayload interface {
	payload.Payload
	IsFinal() bool
}

// finalPayloadWrapper wraps a payload and marks it as final.
type finalPayloadWrapper struct {
	payload.Payload
}

func (f finalPayloadWrapper) IsFinal() bool {
	return true
}

// NewFinalPayload creates a final payload that will be sent with FlagNext|FlagComplete.
func NewFinalPayload(p payload.Payload) payload.Payload {
	return finalPayloadWrapper{Payload: p}
}

type requestChannelSubscriber struct {
	sid uint32
	dc  *DuplexConnection
	rcv flux.Processor
}

func (r *requestChannelSubscriber) OnNext(item payload.Payload) {
	r.dc.sendPayload(r.sid, item, core.FlagNext)
}

func (r *requestChannelSubscriber) OnError(err error) {
	r.dc.writeError(r.sid, err)
}

func (r *requestChannelSubscriber) OnComplete() {
	complete := framing.NewWriteablePayloadFrame(r.sid, nil, nil, core.FlagComplete)
	done := make(chan struct{})
	complete.HandleDone(func() {
		close(done)
	})
	if r.dc.sendFrame(complete) {
		<-done
	}
}

func (r *requestChannelSubscriber) OnSubscribe(ctx context.Context, s rx.Subscription) {
	select {
	case <-ctx.Done():
		r.OnError(reactor.ErrSubscribeCancelled)
	default:
		cb := requestChannelCallback{
			rcv: r.rcv,
			snd: s,
		}
		r.dc.register(r.sid, cb)
		s.Request(1)
	}
}

type respondChannelSubscriber struct {
	sid           uint32
	n             uint32
	dc            *DuplexConnection
	rcv           flux.Processor
	subscribed    chan<- struct{}
	calls         *atomic.Int32
	sentFinalNext atomic.Bool
}

func (r *respondChannelSubscriber) OnNext(next payload.Payload) {
	if _, ok := next.(FinalPayload); ok {
		r.sentFinalNext.Store(true)
		r.OnComplete()
		r.dc.sendPayload(r.sid, next, core.FlagNext|core.FlagComplete)
		return
	}
	r.dc.sendPayload(r.sid, next, core.FlagNext)
}

func (r *respondChannelSubscriber) OnError(err error) {
	if r.calls.Inc() == 2 {
		r.dc.unregister(r.sid)
	}
	r.dc.writeError(r.sid, err)
}

func (r *respondChannelSubscriber) OnComplete() {
	if r.calls.Inc() == 2 {
		r.dc.unregister(r.sid)
	}
	if r.sentFinalNext.Load() {
		return
	}
	complete := framing.NewWriteablePayloadFrame(r.sid, nil, nil, core.FlagComplete)
	done := make(chan struct{})
	complete.HandleDone(func() {
		close(done)
	})
	if r.dc.sendFrame(complete) {
		<-done
	}
}

func (r *respondChannelSubscriber) OnSubscribe(ctx context.Context, s rx.Subscription) {
	select {
	case <-ctx.Done():
		r.OnError(reactor.ErrSubscribeCancelled)
	default:
		cb := respondChannelCallback{
			rcv: r.rcv,
			snd: s,
		}
		r.dc.register(r.sid, cb)
		close(r.subscribed)
		s.Request(ToIntRequestN(r.n))
	}
}
