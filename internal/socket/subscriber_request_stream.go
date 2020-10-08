package socket

import (
	"context"
	"sync"

	"github.com/jjeffcaii/reactor-go"
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/fragmentation"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

var _requestStreamSubscriberPool = sync.Pool{
	New: func() interface{} {
		return new(requestStreamSubscriber)
	},
}

type requestStreamSubscriber struct {
	n         uint32
	sid       uint32
	dc        *DuplexConnection
	receiving fragmentation.HeaderAndPayload
}

func borrowRequestStreamSubscriber(receiving fragmentation.HeaderAndPayload, dc *DuplexConnection, sid uint32, n uint32) rx.Subscriber {
	s := _requestStreamSubscriberPool.Get().(*requestStreamSubscriber)
	s.sid = sid
	s.dc = dc
	s.n = n
	s.receiving = receiving
	return s
}

func returnRequestStreamSubscriber(s rx.Subscriber) {
	actual, ok := s.(*requestStreamSubscriber)
	if !ok {
		return
	}
	common.TryRelease(actual.receiving)
	actual.dc = nil
	actual.receiving = nil
	_requestStreamSubscriberPool.Put(actual)
}

func (r *requestStreamSubscriber) OnNext(payload payload.Payload) {
	r.dc.sendPayload(r.sid, payload, core.FlagNext)
}

func (r *requestStreamSubscriber) OnError(err error) {
	defer func() {
		r.dc.unregister(r.sid)
		returnRequestStreamSubscriber(r)
	}()
	r.dc.writeError(r.sid, err)
}

func (r *requestStreamSubscriber) OnComplete() {
	defer func() {
		r.dc.unregister(r.sid)
		returnRequestStreamSubscriber(r)
	}()
	r.dc.sendFrame(framing.NewWriteablePayloadFrame(r.sid, nil, nil, core.FlagComplete))
}

func (r *requestStreamSubscriber) OnSubscribe(ctx context.Context, subscription rx.Subscription) {
	select {
	case <-ctx.Done():
		r.OnError(reactor.ErrSubscribeCancelled)
	default:
		r.dc.register(r.sid, requestStreamCallbackReverse{su: subscription})
		subscription.Request(int(r.n))
	}
}
