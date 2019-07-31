package flux

import (
	"context"

	"github.com/jjeffcaii/reactor-go/flux"
	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

type Sink interface {
	Next(v payload.Payload)
	Complete()
	Error(e error)
}

type Flux interface {
	rx.Publisher
	Take(n int) Flux
	Filter(rx.FnPredicate) Flux
	DoOnError(rx.FnOnError) Flux
	DoOnNext(rx.FnOnNext) Flux
	DoOnComplete(rx.FnOnComplete) Flux
	DoFinally(rx.FnFinally) Flux
	DoOnRequest(rx.FnOnRequest) Flux
	DoOnSubscribe(rx.FnOnSubscribe) Flux
	Map(fn func(in payload.Payload) payload.Payload) Flux
	SwitchOnFirst(FnSwitchOnFirst) Flux
	SubscribeOn(scheduler.Scheduler) Flux
	Raw() flux.Flux
	BlockFirst(context.Context) (payload.Payload, error)
	BlockLast(context.Context) (payload.Payload, error)
	ToChan(ctx context.Context, cap int) (c <-chan payload.Payload, e <-chan error)
}

type Processor interface {
	Sink
	Flux
}

type Signal interface {
	Value() (payload.Payload, bool)
	Type() rx.SignalType
}

type FnSwitchOnFirst = func(s Signal, f Flux) Flux
