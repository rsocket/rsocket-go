package socket

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/jjeffcaii/reactor-go"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/fragmentation"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"

	uatomic "go.uber.org/atomic"
)

var globalRequestResponseSubscriberPool requestResponseSubscriberPool

var _borrows, _returns, _nexts uatomic.Int64

type requestResponseSubscriberPool struct {
	inner sync.Pool
}

func Statistics() (borrows, releases, nexts int64) {
	return _borrows.Load(), _returns.Load(), _nexts.Load()
}

func (p *requestResponseSubscriberPool) get() *requestResponseSubscriber {
	defer _borrows.Inc()
	if exist, _ := p.inner.Get().(*requestResponseSubscriber); exist != nil {
		return exist
	}
	return &requestResponseSubscriber{}
}

func (p *requestResponseSubscriberPool) put(s *requestResponseSubscriber) {
	defer _returns.Inc()
	if s == nil {
		return
	}
	p.inner.Put(s)
}

type requestResponseSubscriber struct {
	dc        *DuplexConnection
	sid       uint32
	receiving fragmentation.HeaderAndPayload
	sndCnt    int32
}

func borrowRequestResponseSubscriber(dc *DuplexConnection, sid uint32, receiving fragmentation.HeaderAndPayload) rx.Subscriber {
	s := globalRequestResponseSubscriberPool.get()
	s.receiving = receiving
	s.dc = dc
	s.sid = sid
	return s
}

func returnRequestResponseSubscriber(s rx.Subscriber) {
	actual, ok := s.(*requestResponseSubscriber)
	if !ok {
		return
	}
	actual.dc = nil
	actual.receiving = nil
	actual.sndCnt = 0
	globalRequestResponseSubscriberPool.put(actual)
}

func (r *requestResponseSubscriber) OnNext(next payload.Payload) {
	defer _nexts.Inc()
	r.dc.sendPayload(r.sid, next, core.FlagNext|core.FlagComplete)
	atomic.AddInt32(&r.sndCnt, 1)
}

func (r *requestResponseSubscriber) OnError(err error) {
	defer func() {
		r.dc.unregister(r.sid)
		r.finish()
	}()
	r.dc.writeError(r.sid, err)
}

func (r *requestResponseSubscriber) OnComplete() {
	if atomic.AddInt32(&r.sndCnt, 1) == 1 {
		r.dc.sendPayload(r.sid, payload.Empty(), core.FlagComplete)
	}
	r.dc.unregister(r.sid)
	r.finish()
}

func (r *requestResponseSubscriber) OnSubscribe(ctx context.Context, su rx.Subscription) {
	select {
	case <-ctx.Done():
		r.OnError(reactor.ErrSubscribeCancelled)
	default:
		r.dc.register(r.sid, requestResponseCallbackReverse{su: su})
		su.Request(rx.RequestMax)
	}
}

func (r *requestResponseSubscriber) finish() {
	common.TryRelease(r.receiving)
	returnRequestResponseSubscriber(r)
}
