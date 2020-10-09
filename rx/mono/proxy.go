package mono

import (
	"context"
	"sync"
	"time"

	"github.com/jjeffcaii/reactor-go"
	"github.com/jjeffcaii/reactor-go/mono"
	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

var pool *sync.Pool = &sync.Pool{
	New: func() interface{} {
		return newProxy(nil)
	},
}
type proxy struct {
	mono.Mono
}

func getObjFromPool(source mono.Mono) proxy {
	var p = pool.Get().(proxy)
	p.Mono = source
	return p
}

func putObjToPool(p *proxy)  {
	p.Mono = nil
	pool.Put(*p)
}

func newProxy(source mono.Mono) proxy {
	return proxy{source}
}

func deepClone(any reactor.Any) (reactor.Any, error) {
	src := any.(payload.Payload)
	m, _ := src.Metadata()
	return payload.New(common.CloneBytes(src.Data()), common.CloneBytes(m)), nil
}

func (p proxy) Raw() mono.Mono {
	return p.Mono
}

func (p proxy) mustProcessor() mono.Processor {
	m, ok := p.Mono.(mono.Processor)
	if !ok {
		panic(errors.Errorf("require processor but %v", p.Mono))
	}
	putObjToPool(&p)
	return m
}

func (p proxy) Success(v payload.Payload) {
	p.mustProcessor().Success(v)
	putObjToPool(&p)
}

func (p proxy) Error(e error) {
	p.mustProcessor().Error(e)
	putObjToPool(&p)
}

func (p proxy) ToChan(ctx context.Context) (<-chan payload.Payload, <-chan error) {
	value := make(chan payload.Payload, 1)
	err := make(chan error, 1)
	p.subscribeWithChan(ctx, value, err, true)
	putObjToPool(&p)
	return value, err
}

func (p proxy) SubscribeOn(sc scheduler.Scheduler) Mono {
	var pp = getObjFromPool(p.Mono.SubscribeOn(sc))
	putObjToPool(&p)
	return pp
}

func (p proxy) subscribeWithChan(ctx context.Context, valueChan chan<- payload.Payload, errChan chan<- error, autoClose bool) {
	p.Mono.
		DoFinally(func(s reactor.SignalType) {
			if autoClose {
				defer close(valueChan)
				defer close(errChan)
			}
			if s == reactor.SignalTypeCancel {
				errChan <- reactor.ErrSubscribeCancelled
			}
		}).
		Subscribe(
			ctx,
			reactor.OnNext(func(v reactor.Any) error {
				valueChan <- v.(payload.Payload)
				return nil
			}),
			reactor.OnError(func(e error) {
				errChan <- e
			}),
		)
	putObjToPool(&p)
}

func (p proxy) SubscribeWithChan(ctx context.Context, valueChan chan<- payload.Payload, errChan chan<- error) {
	p.subscribeWithChan(ctx, valueChan, errChan, false)
	putObjToPool(&p)
}

func (p proxy) Block(ctx context.Context) (pa payload.Payload, err error) {
	v, err := p.Mono.Map(deepClone).Block(ctx)
	if err != nil {
		return
	}
	if v != nil {
		pa = v.(payload.Payload)
	}
	putObjToPool(&p)
	return
}

func (p proxy) Filter(fn rx.FnPredicate) Mono {
	var pp = getObjFromPool(p.Mono.Filter(func(i reactor.Any) bool {
		return fn(i.(payload.Payload))
	}))
	putObjToPool(&p)
	return pp
}

func (p proxy) Map(transform rx.FnTransform) Mono {
	var pp = getObjFromPool(p.Mono.Map(func(any reactor.Any) (reactor.Any, error) {
		return transform(any.(payload.Payload))
	}))
	putObjToPool(&p)
	return pp
}

func (p proxy) FlatMap(transform func(payload.Payload) Mono) Mono {
	var pp = getObjFromPool(p.Mono.FlatMap(func(any reactor.Any) mono.Mono {
		return transform(any.(payload.Payload)).Raw()
	}))
	putObjToPool(&p)
	return pp
}

func (p proxy) DeepClone() Mono {
	var pp = getObjFromPool(p.Mono.Map(deepClone))
	putObjToPool(&p)
	return pp
}

func (p proxy) DoFinally(fn rx.FnFinally) Mono {
	var pp = getObjFromPool(p.Mono.DoFinally(func(signal reactor.SignalType) {
		fn(rx.SignalType(signal))
	}))
	putObjToPool(&p)
	return pp
}

func (p proxy) DoOnError(fn rx.FnOnError) Mono {
	var pp =  getObjFromPool(p.Mono.DoOnError(func(e error) {
		fn(e)
	}))
	putObjToPool(&p)
	return pp
}
func (p proxy) DoOnSuccess(next rx.FnOnNext) Mono {
	var  pp = getObjFromPool(p.Mono.DoOnNext(func(v reactor.Any) error {
		return next(v.(payload.Payload))
	}))
	putObjToPool(&p)
	return pp
}

func (p proxy) DoOnSubscribe(fn rx.FnOnSubscribe) Mono {
	var pp = getObjFromPool(p.Mono.DoOnSubscribe(func(ctx context.Context, su reactor.Subscription) {
		fn(ctx, su)
	}))
	putObjToPool(&p)
	return pp
}

func (p proxy) DoOnCancel(fn rx.FnOnCancel) Mono {
	var pp = getObjFromPool(p.Mono.DoOnCancel(fn))
	putObjToPool(&p)
	return pp
}

func (p proxy) SwitchIfEmpty(alternative Mono) Mono {
	var pp = getObjFromPool(p.Mono.SwitchIfEmpty(alternative.Raw()))
	putObjToPool(&p)
	return pp
}

func (p proxy) Timeout(timeout time.Duration) Mono {
	var pp = getObjFromPool(p.Mono.Timeout(timeout))
	putObjToPool(&p)
	return pp
}

func (p proxy) Subscribe(ctx context.Context, options ...rx.SubscriberOption) {
	p.SubscribeWith(ctx, rx.NewSubscriber(options...))
	putObjToPool(&p)
}

func (p proxy) SubscribeWith(ctx context.Context, actual rx.Subscriber) {
	var sub reactor.Subscriber
	if actual == rx.EmptySubscriber {
		sub = rx.EmptyRawSubscriber
	} else {
		sub = rx.NewSubscriberFacade(actual)
	}
	p.Mono.SubscribeWith(ctx, sub)
	putObjToPool(&p)
}
