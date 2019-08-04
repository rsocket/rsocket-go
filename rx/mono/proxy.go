package mono

import (
	"context"

	"github.com/jjeffcaii/reactor-go"
	"github.com/jjeffcaii/reactor-go/mono"
	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

type proxy struct {
	mono.Mono
}

func (p proxy) Raw() mono.Mono {
	return p.Mono
}

func (p proxy) mustProcessor() mono.Processor {
	m, ok := p.Mono.(mono.Processor)
	if !ok {
		panic(errors.Errorf("require processor but %v", p.Mono))
	}
	return m
}

func (p proxy) Success(v payload.Payload) {
	p.mustProcessor().Success(v)
}

func (p proxy) Error(e error) {
	p.mustProcessor().Error(e)
}

func (p proxy) SubscribeOn(sc scheduler.Scheduler) Mono {
	return newProxy(p.Mono.SubscribeOn(sc))
}

func (p proxy) Block(ctx context.Context) (pa payload.Payload, err error) {
	v, err := p.Mono.Block(ctx)
	if err != nil {
		return
	}
	pa = v.(payload.Payload)
	return
}

func (p proxy) Filter(fn rx.FnPredicate) Mono {
	return newProxy(p.Mono.Filter(func(i interface{}) bool {
		return fn(i.(payload.Payload))
	}))
}

func (p proxy) DoFinally(fn rx.FnFinally) Mono {
	return newProxy(p.Mono.DoFinally(func(signal rs.SignalType) {
		fn(rx.SignalType(signal))
	}))
}

func (p proxy) DoOnError(fn rx.FnOnError) Mono {
	return newProxy(p.Mono.DoOnError(func(e error) {
		fn(e)
	}))
}
func (p proxy) DoOnSuccess(next rx.FnOnNext) Mono {
	return newProxy(p.Mono.DoOnNext(func(v interface{}) {
		next(v.(payload.Payload))
	}))
}

func (p proxy) DoOnSubscribe(fn rx.FnOnSubscribe) Mono {
	return newProxy(p.Mono.DoOnSubscribe(func(su rs.Subscription) {
		fn(su)
	}))
}

func (p proxy) DoOnCancel(fn rx.FnOnCancel) Mono {
	return newProxy(p.Mono.DoOnCancel(fn))
}

func (p proxy) SwitchIfEmpty(alternative Mono) Mono {
	return newProxy(p.Mono.SwitchIfEmpty(alternative.Raw()))
}

func (p proxy) Subscribe(ctx context.Context, options ...rx.SubscriberOption) {
	p.SubscribeWith(ctx, rx.NewSubscriber(options...))
}

func (p proxy) SubscribeWith(ctx context.Context, actual rx.Subscriber) {
	var sub rs.Subscriber
	if actual == rx.EmptySubscriber {
		sub = rx.EmptyRawSubscriber
	} else {
		sub = rs.NewSubscriber(
			rs.OnNext(func(v interface{}) {
				actual.OnNext(v.(payload.Payload))
			}),
			rs.OnComplete(func() {
				actual.OnComplete()
			}),
			rs.OnSubscribe(func(su rs.Subscription) {
				actual.OnSubscribe(su)
			}),
			rs.OnError(func(e error) {
				actual.OnError(e)
			}),
		)
	}
	p.Mono.SubscribeWith(ctx, sub)
}

func newProxy(source mono.Mono) proxy {
	return proxy{source}
}
