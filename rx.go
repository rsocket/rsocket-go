package rsocket

import (
	"context"
	"sync"
)

type Consumer = func(ctx context.Context, item Payload)
type OnFinally = func(ctx context.Context)

type Disposable interface {
	Dispose()
}

type Publisher interface {
	Subscribe(ctx context.Context, consumer Consumer) Disposable
}

type Mono interface {
	Publisher
	DoOnNext(onNext Consumer) Mono
	DoOnSuccess(onSuccess Consumer) Mono
	DoFinally(fn OnFinally) Mono
	SubscribeOn(scheduler Scheduler) Publisher
}

type Flux interface {
	Publisher
	DoOnNext(onNext Consumer) Flux
	DoFinally(fn OnFinally) Flux
	SubscribeOn(scheduler Scheduler) Publisher
}

type Emitter interface {
	Next(item Payload)
	Complete()
	Error(e error)
}

type MonoEmitter interface {
	Success(payload Payload)
	Error(e error)
}

func NewMono(fn func(emitter MonoEmitter)) Mono {
	return &implMono{
		ob:         fn,
		done:       make(chan struct{}),
		schedulerC: ElasticScheduler(),
		schedulerS: ImmediateScheduler(),
		onFinally:  make([]OnFinally, 0),
		handlers:   make([]Consumer, 0),
		once:       &sync.Once{},
	}
}

func JustMono(payload Payload) Mono {
	return NewMono(func(emitter MonoEmitter) {
		emitter.Success(payload)
	})
}
