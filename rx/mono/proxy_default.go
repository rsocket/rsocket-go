package mono

import (
	"context"
	"time"

	"github.com/jjeffcaii/reactor-go"
	"github.com/jjeffcaii/reactor-go/mono"
	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

type proxy struct {
	mono.Mono
}

func newProxy(source mono.Mono) proxy {
	return proxy{source}
}

func (p proxy) Raw() mono.Mono {
	return p.Mono
}

func (p proxy) Success(v payload.Payload) {
	mustProcessor(p.Mono).Success(v)
}

func (p proxy) Error(e error) {
	mustProcessor(p.Mono).Error(e)
}

func (p proxy) ToChan(ctx context.Context) (<-chan payload.Payload, <-chan error) {
	return toChan(ctx, p.Mono)
}

func (p proxy) SubscribeOn(sc scheduler.Scheduler) Mono {
	return newProxy(p.Mono.SubscribeOn(sc))
}

func (p proxy) SubscribeWithChan(ctx context.Context, valueChan chan<- payload.Payload, errChan chan<- error) {
	subscribeWithChan(ctx, p.Mono, valueChan, errChan, false)
}

func (p proxy) Block(ctx context.Context) (pa payload.Payload, err error) {
	v, err := p.Mono.Map(deepClone).Block(ctx)
	if err != nil {
		return
	}
	if v != nil {
		pa = v.(payload.Payload)
	}
	return
}

func (p proxy) Filter(fn rx.FnPredicate) Mono {
	return newProxy(p.Mono.Filter(func(i reactor.Any) bool {
		return fn(i.(payload.Payload))
	}))
}

func (p proxy) Map(transform rx.FnTransform) Mono {
	return newProxy(p.Mono.Map(func(any reactor.Any) (reactor.Any, error) {
		return transform(any.(payload.Payload))
	}))
}

func (p proxy) FlatMap(transform func(payload.Payload) Mono) Mono {
	return newProxy(p.Mono.FlatMap(func(any reactor.Any) mono.Mono {
		return transform(any.(payload.Payload)).Raw()
	}))
}

func (p proxy) DeepClone() Mono {
	return newProxy(p.Mono.Map(deepClone))
}

func (p proxy) DoFinally(fn rx.FnFinally) Mono {
	return newProxy(p.Mono.DoFinally(func(signal reactor.SignalType) {
		fn(rx.SignalType(signal))
	}))
}

func (p proxy) DoOnError(fn rx.FnOnError) Mono {
	return newProxy(p.Mono.DoOnError(func(e error) {
		fn(e)
	}))
}
func (p proxy) DoOnSuccess(next rx.FnOnNext) Mono {
	return newProxy(p.Mono.DoOnNext(func(v reactor.Any) error {
		return next(v.(payload.Payload))
	}))
}

func (p proxy) DoOnSubscribe(fn rx.FnOnSubscribe) Mono {
	return newProxy(p.Mono.DoOnSubscribe(func(ctx context.Context, su reactor.Subscription) {
		fn(ctx, su)
	}))
}

func (p proxy) DoOnCancel(fn rx.FnOnCancel) Mono {
	return newProxy(p.Mono.DoOnCancel(fn))
}

func (p proxy) SwitchIfEmpty(alternative Mono) Mono {
	return newProxy(p.Mono.SwitchIfEmpty(alternative.Raw()))
}

func (p proxy) Timeout(timeout time.Duration) Mono {
	return newProxy(p.Mono.Timeout(timeout))
}

func (p proxy) Subscribe(ctx context.Context, options ...rx.SubscriberOption) {
	p.SubscribeWith(ctx, rx.NewSubscriber(options...))
}

func (p proxy) SubscribeWith(ctx context.Context, actual rx.Subscriber) {
	var sub reactor.Subscriber
	if actual == rx.EmptySubscriber {
		sub = rx.EmptyRawSubscriber
	} else {
		sub = rx.NewSubscriberFacade(actual)
	}
	p.Mono.SubscribeWith(ctx, sub)
}
