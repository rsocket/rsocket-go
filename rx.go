package rsocket

import (
	"context"
	"sync"
)

type Consumer = func(ctx context.Context, item Payload)
type OnCancel = func(ctx context.Context)
type OnExhaust = func(ctx context.Context)
type OnFinally = func(ctx context.Context, sig SignalType)
type OnError = func(ctx context.Context, err error)
type OnComplete = func(ctx context.Context)
type OnSubscribe = func(ctx context.Context)

const (
	SigReady SignalType = iota
	SigSuccess
	SigError
	SigCancel
)

type SignalType uint8

type Disposable interface {
	//IsDisposed() bool
	Dispose()
}

type Publisher interface {
	Subscribe(ctx context.Context, consumer Consumer) Disposable
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
	Publisher
	LimitRate(n uint32) Flux
	DoOnCancel(onCancel OnCancel) Flux
	DoOnNext(onNext Consumer) Flux
	DoOnError(onError OnError) Flux
	DoOnComplete(onSuccess OnComplete) Flux
	DoFinally(fn OnFinally) Flux
	DoOnSubscribe(subscribe OnSubscribe) Flux
	SubscribeOn(scheduler Scheduler) Publisher
	onAfterSubscribe(fn Consumer) Flux
	limiter() (*rateLimiter, bool)
	onExhaust(onExhaust OnExhaust) Flux
	resetLimit(n uint32)
}

type FluxEmitter interface {
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

func NewFlux(create func(ctx context.Context, emitter FluxEmitter)) Flux {
	if create == nil {
		panic(ErrInvalidEmitter)
	}
	f := newImplFlux()
	f.creation = create
	return f
}
