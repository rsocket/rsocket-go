package flux

import (
	"context"

	rs "github.com/jjeffcaii/reactor-go"
	"github.com/jjeffcaii/reactor-go/flux"
	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

type proxy struct {
	flux.Flux
}

func (p proxy) Raw() flux.Flux {
	return p.Flux
}

func (p proxy) mustProcessor() flux.Processor {
	proc, ok := p.Flux.(flux.Processor)
	if !ok {
		panic(errors.New("require flux.Processor"))
	}
	return proc
}

func (p proxy) Next(v payload.Payload) {
	p.mustProcessor().Next(v)
}

func (p proxy) Complete() {
	p.mustProcessor().Complete()
}

func (p proxy) Error(e error) {
	p.mustProcessor().Error(e)
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
	return newProxy(p.Flux.DoOnNext(func(v interface{}) {
		fn(v.(payload.Payload))
	}))
}

func (p proxy) BlockLast(ctx context.Context) (last payload.Payload, err error) {
	done := make(chan struct{})
	sub := rs.NewSubscriber(
		rs.OnNext(func(v interface{}) {
			if last != nil {
				last.Release()
			}
			last = v.(payload.Payload)
		}),
		rs.OnError(func(e error) {
			err = e
		}),
	)
	p.Flux.
		DoFinally(func(s rs.SignalType) {
			if s == rs.SignalTypeCancel {
				err = rs.ErrSubscribeCancelled
			}
			close(done)
		}).
		SubscribeWith(ctx, sub)
	<-done
	if err != nil && last != nil {
		last.Release()
		last = nil
	}
	// To prevent bytebuff leak, clone it.
	origin := last
	last = payload.Clone(origin)
	origin.Release()
	return
}

func (p proxy) DoOnSubscribe(fn rx.FnOnSubscribe) Flux {
	return newProxy(p.Flux.DoOnSubscribe(func(su rs.Subscription) {
		fn(su)
	}))
}

func (p proxy) DoOnRequest(fn rx.FnOnRequest) Flux {
	return newProxy(p.Flux.DoOnRequest(fn))
}

func (p proxy) DoFinally(fn rx.FnFinally) Flux {
	return newProxy(p.Flux.DoFinally(func(s rs.SignalType) {
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
	var sub rs.Subscriber
	if s == rx.EmptySubscriber {
		sub = rx.EmptyRawSubscriber
	} else {
		sub = rs.NewSubscriber(
			rs.OnNext(func(v interface{}) {
				s.OnNext(v.(payload.Payload))
			}),
			rs.OnError(func(e error) {
				s.OnError(e)
			}),
			rs.OnComplete(func() {
				s.OnComplete()
			}),
			rs.OnSubscribe(func(su rs.Subscription) {
				s.OnSubscribe(su)
			}),
		)
	}
	p.Flux.SubscribeWith(ctx, sub)
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
