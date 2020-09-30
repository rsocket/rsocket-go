package flux

import (
	"context"

	"github.com/jjeffcaii/reactor-go"
	"github.com/jjeffcaii/reactor-go/flux"
	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

type proxy struct {
	flux.Flux
}

func (p proxy) Raw() flux.Flux {
	return p.Flux
}

func (p proxy) Next(v payload.Payload) {
	p.mustProcessor().Next(v)
}

func (p proxy) Map(fn func(in payload.Payload) (payload.Payload, error)) Flux {
	return newProxy(p.Flux.Map(func(i reactor.Any) (reactor.Any, error) {
		return fn(i.(payload.Payload))
	}))
}

func (p proxy) Complete() {
	p.mustProcessor().Complete()
}

func (p proxy) Error(e error) {
	p.mustProcessor().Error(e)
}

func (p proxy) Take(n int) Flux {
	return newProxy(p.Flux.Take(n))
}

func (p proxy) Filter(fn rx.FnPredicate) Flux {
	return newProxy(p.Flux.Filter(func(i interface{}) bool {
		return fn(i.(payload.Payload))
	}))
}

func (p proxy) DoOnComplete(fn rx.FnOnComplete) Flux {
	return newProxy(p.Flux.DoOnComplete(fn))
}

func (p proxy) DoOnError(fn rx.FnOnError) Flux {
	return newProxy(p.Flux.DoOnError(fn))
}

func (p proxy) DoOnNext(fn rx.FnOnNext) Flux {
	return newProxy(p.Flux.DoOnNext(func(v reactor.Any) error {
		return fn(v.(payload.Payload))
	}))
}

func (p proxy) ToChan(ctx context.Context, cap int) (<-chan payload.Payload, <-chan error) {
	if cap < 1 {
		cap = 1
	}
	ch := make(chan payload.Payload, cap)
	err := make(chan error, 1)
	p.Flux.
		DoFinally(func(s reactor.SignalType) {
			defer func() {
				close(ch)
				close(err)
			}()
			if s == reactor.SignalTypeCancel {
				err <- reactor.ErrSubscribeCancelled
			}
		}).
		SubscribeWithChan(ctx, ch, err)
	return ch, err
}

func (p proxy) BlockFirst(ctx context.Context) (first payload.Payload, err error) {
	v, err := p.Flux.BlockFirst(ctx)
	if err != nil {
		return
	}
	if v != nil {
		first = v.(payload.Payload)
	}
	return
}

func (p proxy) BlockLast(ctx context.Context) (last payload.Payload, err error) {
	v, err := p.Flux.BlockLast(ctx)
	if err != nil {
		return
	}
	if v == nil {
		return
	}
	last = v.(payload.Payload)
	return
}

func (p proxy) SubscribeWithChan(ctx context.Context, payloads chan<- payload.Payload, err chan<- error) {
	p.Flux.SubscribeWithChan(ctx, payloads, err)
}

func (p proxy) BlockSlice(ctx context.Context) (results []payload.Payload, err error) {
	done := make(chan struct{})
	p.Flux.
		DoFinally(func(s reactor.SignalType) {
			defer close(done)
			if s == reactor.SignalTypeCancel {
				err = reactor.ErrSubscribeCancelled
			}
		}).
		Subscribe(
			ctx,
			reactor.OnNext(func(v reactor.Any) error {
				results = append(results, v.(payload.Payload))
				return nil
			}),
			reactor.OnError(func(e error) {
				err = e
			}),
		)
	<-done
	return
}

func (p proxy) DoOnSubscribe(fn rx.FnOnSubscribe) Flux {
	return newProxy(p.Flux.DoOnSubscribe(func(ctx context.Context, su reactor.Subscription) {
		fn(ctx, su)
	}))
}

func (p proxy) DoOnRequest(fn rx.FnOnRequest) Flux {
	return newProxy(p.Flux.DoOnRequest(fn))
}

func (p proxy) DoFinally(fn rx.FnFinally) Flux {
	return newProxy(p.Flux.DoFinally(func(s reactor.SignalType) {
		fn(rx.SignalType(s))
	}))
}

func (p proxy) SwitchOnFirst(fn FnSwitchOnFirst) Flux {
	return newProxy(p.Flux.SwitchOnFirst(func(s flux.Signal, f flux.Flux) flux.Flux {
		return fn(newSignal(s), newProxy(f)).Raw()
	}))
}

func (p proxy) SubscribeOn(sc scheduler.Scheduler) Flux {
	return newProxy(p.Flux.SubscribeOn(sc))
}

func (p proxy) Subscribe(ctx context.Context, options ...rx.SubscriberOption) {
	p.SubscribeWith(ctx, rx.NewSubscriber(options...))
}

func (p proxy) SubscribeWith(ctx context.Context, s rx.Subscriber) {
	var sub reactor.Subscriber
	if s == rx.EmptySubscriber {
		sub = rx.EmptyRawSubscriber
	} else {
		sub = rx.NewSubscriberFacade(s)
	}
	p.Flux.SubscribeWith(ctx, sub)
}

func (p proxy) mustProcessor() flux.Processor {
	processor, ok := p.Flux.(flux.Processor)
	if !ok {
		panic("require flux.Processor")
	}
	return processor
}

type sinkProxy struct {
	flux.Sink
}

func (s sinkProxy) Next(v payload.Payload) {
	s.Sink.Next(v)
}

type pxSignal struct {
	flux.Signal
}

func (p pxSignal) Value() (v payload.Payload, ok bool) {
	found, ok := p.Signal.Value()
	if ok {
		v = found.(payload.Payload)
	}
	return
}

func (p pxSignal) Type() rx.SignalType {
	return rx.SignalType(p.Signal.Type())
}

func newSignal(origin flux.Signal) pxSignal {
	return pxSignal{origin}
}

func newProxySink(sink flux.Sink) sinkProxy {
	return sinkProxy{sink}
}

func newProxy(f flux.Flux) proxy {
	return proxy{f}
}
