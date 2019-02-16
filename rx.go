package rsocket

import (
	"context"
	"sync"
)

type Consumer = func(ctx context.Context, item Payload)
type consumerIndexed = func(ctx context.Context, item Payload, i int)
type OnCancel = func(ctx context.Context)
type OnFinally = func(ctx context.Context)
type OnError = func(ctx context.Context, err error)
type OnComplete = func(ctx context.Context)

type Disposable interface {
	//IsDisposed() bool
	Dispose()
}

type Publisher interface {
	Subscribe(ctx context.Context, consumer Consumer) Disposable
}

type indexedPublisher interface {
	Publisher
	subscribeIndexed(ctx context.Context, indexed consumerIndexed) Disposable
}

type Mono interface {
	Publisher
	DoOnCancel(onCancel OnCancel) Mono
	DoOnNext(onNext Consumer) Mono
	DoOnSuccess(onSuccess Consumer) Mono
	DoOnError(onError OnError) Mono
	DoFinally(fn OnFinally) Mono
	SubscribeOn(scheduler Scheduler) Publisher
	onAfterSubscribe(fn Consumer) Mono
}

type Flux interface {
	indexedPublisher
	DoOnCancel(onCancel OnCancel) Flux
	DoOnNext(onNext Consumer) Flux
	DoOnError(onError OnError) Flux
	DoOnComplete(onSuccess OnComplete) Flux
	DoFinally(fn OnFinally) Flux
	SubscribeOn(scheduler Scheduler) indexedPublisher
	onAfterSubscribe(fn Consumer) Flux
}

type Emitter interface {
	Next(payload Payload)
	Error(err error)
	Complete()
}

type MonoEmitter interface {
	Success(payload Payload)
	Error(err error)
}

type MonoCreate = func(ctx context.Context, emitter MonoEmitter)

func NewMono(create MonoCreate) Mono {
	return &implMono{
		done:     make(chan struct{}),
		ss:       ImmediateScheduler(),
		locker:   &sync.Mutex{},
		init:     create,
		sig:      SigReady,
		handlers: newRxFuncStore(),
	}
}

func JustMono(payload Payload) Mono {
	return &implMonoSingle{
		value:    payload,
		ss:       ImmediateScheduler(),
		handlers: newRxFuncStore(),
		locker:   &sync.Mutex{},
	}
}
