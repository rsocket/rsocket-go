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
)

var _requestResponseSubscriberRefs int32

func CountRequestResponseRef() int32 {
	return atomic.LoadInt32(&_requestResponseSubscriberRefs)
}

var _requestResponseSubscriberPool = sync.Pool{
	New: func() interface{} {
		return new(requestResponseSubscriber)
	},
}

type requestResponseSubscriber struct {
	dc        *DuplexConnection
	sid       uint32
	receiving fragmentation.HeaderAndPayload
}

func borrowRequestResponseSubscriber(dc *DuplexConnection, sid uint32, receiving fragmentation.HeaderAndPayload) rx.Subscriber {
	defer atomic.AddInt32(&_requestResponseSubscriberRefs, 1)
	s := _requestResponseSubscriberPool.Get().(*requestResponseSubscriber)
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
	defer atomic.AddInt32(&_requestResponseSubscriberRefs, -1)
	actual.dc = nil
	actual.receiving = nil
	_requestResponseSubscriberPool.Put(actual)
}

func (r *requestResponseSubscriber) OnNext(payload payload.Payload) {
	r.dc.sendPayload(r.sid, payload, core.FlagNext|core.FlagComplete)
}

func (r *requestResponseSubscriber) OnError(err error) {
	defer func() {
		r.dc.unregister(r.sid)
		r.finish()
	}()
	r.dc.writeError(r.sid, err)
}

func (r *requestResponseSubscriber) OnComplete() {
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
