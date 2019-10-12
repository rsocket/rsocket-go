package mono

import (
	"context"

	"github.com/jjeffcaii/reactor-go/mono"
	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

type Mono interface {
	rx.Publisher
	Filter(rx.FnPredicate) Mono
	DoFinally(rx.FnFinally) Mono
	DoOnError(rx.FnOnError) Mono
	DoOnSuccess(rx.FnOnNext) Mono
	DoOnCancel(rx.FnOnCancel) Mono
	DoOnSubscribe(rx.FnOnSubscribe) Mono
	SubscribeOn(scheduler.Scheduler) Mono
	Block(context.Context) (payload.Payload, error)
	SwitchIfEmpty(alternative Mono) Mono
	Raw() mono.Mono
	// ToChan subscribe Mono and puts items into a chan.
	// It also puts errors into another chan.
	ToChan(ctx context.Context) (c <-chan payload.Payload, e <-chan error)
}

type Sink interface {
	Success(payload.Payload)
	Error(error)
}

type Processor interface {
	Sink
	Mono
}
