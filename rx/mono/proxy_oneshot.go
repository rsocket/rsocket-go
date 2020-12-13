package mono

import (
	"context"
	"sync"
	"time"

	"github.com/jjeffcaii/reactor-go"
	"github.com/jjeffcaii/reactor-go/mono"
	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

var _oneshotProxyPool = sync.Pool{
	New: func() interface{} {
		return new(oneshotProxy)
	},
}

type oneshotProxy struct {
	mono.Mono
}

func borrowOneshotProxy(origin mono.Mono) *oneshotProxy {
	o := _oneshotProxyPool.Get().(*oneshotProxy)
	o.Mono = origin
	return o
}

func returnOneshotProxy(o *oneshotProxy) (raw mono.Mono) {
	raw, o.Mono = o.Mono, nil
	_oneshotProxyPool.Put(o)
	return
}

func (o *oneshotProxy) SubscribeWith(ctx context.Context, s rx.Subscriber) {
	var sub reactor.Subscriber
	if s == rx.EmptySubscriber {
		sub = rx.EmptyRawSubscriber
	} else {
		sub = rx.NewSubscriberFacade(s)
	}
	returnOneshotProxy(o).SubscribeWith(ctx, sub)
}

func (o *oneshotProxy) Subscribe(ctx context.Context, options ...rx.SubscriberOption) {
	o.SubscribeWith(ctx, rx.NewSubscriber(options...))
}

func (o *oneshotProxy) Filter(predicate rx.FnPredicate) Mono {
	o.Mono = o.Mono.Filter(func(any reactor.Any) bool {
		return predicate(any.(payload.Payload))
	})
	return o
}

func (o *oneshotProxy) Map(transform rx.FnTransform) Mono {
	o.Mono = o.Mono.Map(func(any reactor.Any) (reactor.Any, error) {
		return transform(any.(payload.Payload))
	})
	return o
}

func (o *oneshotProxy) FlatMap(f func(payload.Payload) Mono) Mono {
	o.Mono = o.Mono.FlatMap(func(any reactor.Any) mono.Mono {
		return f(any.(payload.Payload)).Raw()
	})
	return o
}

func (o *oneshotProxy) DoFinally(finally rx.FnFinally) Mono {
	o.Mono = o.Mono.DoFinally(func(s reactor.SignalType) {
		finally(rx.SignalType(s))
	})
	return o
}

func (o *oneshotProxy) DoOnError(onError rx.FnOnError) Mono {
	o.Mono = o.Mono.DoOnError(func(e error) {
		onError(e)
	})
	return o
}

func (o *oneshotProxy) DoOnSuccess(next rx.FnOnNext) Mono {
	o.Mono = o.Mono.DoOnNext(func(v reactor.Any) error {
		return next(v.(payload.Payload))
	})
	return o
}

func (o *oneshotProxy) DoOnCancel(cancel rx.FnOnCancel) Mono {
	o.Mono = o.Mono.DoOnCancel(cancel)
	return o
}

func (o *oneshotProxy) DoOnSubscribe(subscribe rx.FnOnSubscribe) Mono {
	o.Mono = o.Mono.DoOnSubscribe(subscribe)
	return o
}

func (o *oneshotProxy) SubscribeOn(scheduler scheduler.Scheduler) Mono {
	o.Mono = o.Mono.SubscribeOn(scheduler)
	return o
}

func (o *oneshotProxy) SubscribeWithChan(ctx context.Context, valueChan chan<- payload.Payload, errChan chan<- error) {
	subscribeWithChan(ctx, returnOneshotProxy(o), valueChan, errChan, false)
}

func (o *oneshotProxy) BlockUnsafe(ctx context.Context) (payload.Payload, ReleaseFunc, error) {
	v, err := toBlock(ctx, returnOneshotProxy(o))
	if err != nil {
		return nil, nil, err
	}
	r, ok := v.(common.Releasable)
	if !ok {
		return v, noopRelease, nil
	}
	return v, r.Release, nil
}

func (o *oneshotProxy) Block(ctx context.Context) (payload.Payload, error) {
	v, r, err := o.BlockUnsafe(ctx)
	if err != nil {
		return nil, err
	}
	defer r()
	if v == nil {
		return nil, nil
	}
	return payload.Clone(v), nil
}

func (o *oneshotProxy) SwitchIfEmpty(alternative Mono) Mono {
	o.Mono = o.Mono.SwitchIfEmpty(unpackRawPublisher(alternative))
	return o
}

func (o *oneshotProxy) SwitchIfError(alternative func(error) Mono) Mono {
	o.Mono = o.Mono.SwitchIfError(func(err error) mono.Mono {
		return unpackRawPublisher(alternative(err))
	})
	return o
}

func (o *oneshotProxy) SwitchValueIfError(alternative payload.Payload) Mono {
	o.Mono = o.Mono.SwitchValueIfError(alternative)
	return o
}
func (o *oneshotProxy) ZipWith(alternative Mono, cmb Combinator2) Mono {
	return Zip(o, alternative).ToMonoOneshot(func(item rx.Tuple) (payload.Payload, error) {
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

func (o *oneshotProxy) Raw() mono.Mono {
	return o.Mono
}

func (o *oneshotProxy) ToChan(ctx context.Context) (c <-chan payload.Payload, e <-chan error) {
	return toChan(ctx, o.Mono)
}

func (o *oneshotProxy) Timeout(timeout time.Duration) Mono {
	o.Mono = o.Mono.Timeout(timeout)
	return o
}
