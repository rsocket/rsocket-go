package mono

import (
	"context"
	"time"

	"github.com/jjeffcaii/reactor-go"
	"github.com/jjeffcaii/reactor-go/mono"
	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

type proxy struct {
	mono.Mono
}

func newProxy(source mono.Mono) proxy {
	return proxy{source}
}

func noopRelease() {
}

func (p proxy) Raw() mono.Mono {
	return p.Mono
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

func (p proxy) BlockUnsafe(ctx context.Context) (payload.Payload, ReleaseFunc, error) {
	v, err := toBlock(ctx, p.Mono)
	if err != nil {
		return nil, nil, err
	}
	r, ok := v.(common.Releasable)
	if !ok {
		return v, noopRelease, nil
	}
	return v, r.Release, nil
}

func (p proxy) Block(ctx context.Context) (payload.Payload, error) {
	v, r, err := p.BlockUnsafe(ctx)
	if err != nil {
		return nil, err
	}
	defer r()
	if v == nil {
		return nil, nil
	}
	if _, ok := v.(common.Releasable); ok {
		return payload.Clone(v), nil
	}
	return v, nil
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
	return newProxy(p.Mono.SwitchIfEmpty(unpackRawPublisher(alternative)))
}

func (p proxy) SwitchIfError(alternative func(error) Mono) Mono {
	return newProxy(p.Mono.SwitchIfError(func(err error) mono.Mono {
		return unpackRawPublisher(alternative(err))
	}))
}

func (p proxy) SwitchValueIfError(alternative payload.Payload) Mono {
	return newProxy(p.Mono.SwitchValueIfError(alternative))
}

func (p proxy) Timeout(timeout time.Duration) Mono {
	return newProxy(p.Mono.Timeout(timeout))
}

func (p proxy) ZipWith(alternative Mono, cmb Combinator2) Mono {
	return Zip(p, alternative).ToMono(func(item rx.Tuple) (payload.Payload, error) {
		first, err := convertItem(item[0])
		if err != nil {
			return nil, err
		}
		second, err := convertItem(item[1])
		if err != nil {
			return nil, err
		}
		return cmb(first, second)
	})
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
