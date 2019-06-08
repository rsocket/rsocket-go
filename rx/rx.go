package rx

import (
	"context"
	"errors"

	"github.com/rsocket/rsocket-go/payload"
)

var (
	errWrongSignal = errors.New("wrong current signal")
)

const (
	signalDefault SignalType = iota
	// SignalComplete indicated that subscriber was completed.
	SignalComplete
	// SignalCancel indicates that subscriber was cancelled.
	SignalCancel
	// SignalError indicates that subscriber has some faults.
	SignalError
)

type (
	// FnConsumer is alias of consumer function.
	FnConsumer = func(ctx context.Context, elem payload.Payload)
	// FnOnComplete is alias of `OnComplete` handler.
	FnOnComplete = func(ctx context.Context)
	// FnOnNext is alias of `OnNext` handler.
	FnOnNext = func(ctx context.Context, s Subscription, elem payload.Payload)
	// FnOnCancel is alias of `OnCancel` handler.
	FnOnCancel = func(ctx context.Context)
	// FnOnSubscribe is alias of `OnSubscribe` handler.
	FnOnSubscribe = func(ctx context.Context, s Subscription)
	// FnOnRequest is alias of `OnRequest` handler.
	FnOnRequest = func(ctx context.Context, n int)
	// FnOnError is alias of `OnError` handler.
	FnOnError = func(ctx context.Context, err error)
	// FnOnFinally is alias of `OnFinally` handler.
	FnOnFinally = func(ctx context.Context, st SignalType)
)

type (
	// SignalType is the signal of reactive events like `OnNext`, `OnComplete`, `OnCancel` and `OnError`.
	SignalType int8

	// Disposable is a disposable resource.
	Disposable interface {
		// Dispose dispose current resource.
		Dispose()
		// IsDisposed returns true if it has been disposed.
		IsDisposed() bool
	}

	// Publisher is a provider of a potentially unbounded number of sequenced elements, \
	// publishing them according to the demand received from its Subscriber(s).
	Publisher interface {
		// Subscribe subscribe elements from a publisher, returns a Disposable.
		// You can add some custome options.
		// Using `OnSubscribe`, `OnNext`, `OnComplete` and `OnError` as handler wrapper.
		Subscribe(ctx context.Context, ops ...OptSubscribe) Disposable
	}

	// Subscriber consume elements from a Publisher and handle events.
	Subscriber interface {
		// OnSubscribe handle event when subscribe begin.
		OnSubscribe(ctx context.Context, s Subscription)
		// OnNext handle event when a new element produced.
		OnNext(ctx context.Context, s Subscription, elem payload.Payload)
		// OnComplete handle event when subscribe finish.
		OnComplete(ctx context.Context)
		// OnError handle event when an error occurred。
		OnError(ctx context.Context, err error)
	}

	// Subscription means a Subscrber's subscription.
	Subscription interface {
		// Request pull next n elements. (It was used for FlowControl)
		// When you call it, subscriber will emit `OnRequest` event and you can use `DoOnRequest` catch it.
		Request(n int)
		// Cancel cancel the current subscriber.
		// Subscribers will emit `OnCancel` event and you can use `DoOnCancel` catch it.
		Cancel()
		// N returns current N in queue.
		N() int
	}

	// Processor process publisher and subscriber.
	Processor interface {
		Publisher
		Subscriber
	}

	// Producer produce elements as you wish.
	Producer interface {
		// Next append next element.
		Next(elem payload.Payload) error
		// Error means some bad things happened.
		Error(err error)
		// Complete means production completed.
		Complete()
	}

	// MonoProducer likes Producer, but it produce single element.
	MonoProducer interface {
		// Success append payload.
		Success(elem payload.Payload) error
		// Error means some bad things happened.
		Error(err error)
	}

	// Mono completes successfully by emitting an element, or with an error.
	Mono interface {
		Publisher
		// DoAfterSuccess register handler after emitting element successfully.
		DoAfterSuccess(fn FnConsumer) Mono
		// DoOnSubscribe register handler on subscribe begin.
		DoOnSubscribe(fn FnOnSubscribe) Mono
		// DoOnSuccess register handler when emitting element successfully.
		DoOnSuccess(fn FnOnNext) Mono
		// DoOnError register handler when an exception occurs.
		DoOnError(fn FnOnError) Mono
		// DoOnCancel register handler when Mono was canceled.
		DoOnCancel(fn FnOnCancel) Mono
		// DoFinally register handler when Mono was terminated.
		// DoFinally will definitely be executed.
		DoFinally(fn FnOnFinally) Mono
		// SubscribeOn specify scheduler for subscriber.
		SubscribeOn(s Scheduler) Mono
		// PublishOn specify scheduler for publisher.
		PublishOn(s Scheduler) Mono
	}

	// Flux emits 0 to N elements, and then completes (successfully or with an error).
	Flux interface {
		Publisher
		// LimitRate limits the number of elements in batches.
		LimitRate(n int) Flux
		// DoOnRequest register handler when subsccriber request more elements.
		DoOnRequest(fn FnOnRequest) Flux
		// DoOnSubscribe register handler when subscribe begin.
		DoOnSubscribe(fn FnOnSubscribe) Flux
		// DoOnNext register handler when emitting next element.
		DoOnNext(fn FnOnNext) Flux
		// DoOnNext register handler after emitting next element.
		DoAfterNext(fn FnConsumer) Flux
		// DoOnComplete register handler when Flux was completed.
		DoOnComplete(fn FnOnComplete) Flux
		// DoOnError register handler when an exception occurs.
		DoOnError(fn FnOnError) Flux
		// DoOnCancel register handler when Mono was canceled.
		DoOnCancel(fn FnOnCancel) Flux
		// DoFinally register handler when Mono was terminated.
		// DoFinally will definitely be executed.
		DoFinally(fn FnOnFinally) Flux
		// SubscribeOn specify scheduler for subscriber.
		SubscribeOn(s Scheduler) Flux
		// PublishOn specify scheduler for publisher.
		PublishOn(s Scheduler) Flux
	}

	// OptSubscribe is option of subscribe.
	OptSubscribe func(*hooks)
)

// OnNext sets handler for OnNext.
func OnNext(fn FnOnNext) OptSubscribe {
	return func(hooks *hooks) {
		hooks.DoOnNext(fn)
	}
}

// OnComplete sets handler for OnComplete.
func OnComplete(fn FnOnComplete) OptSubscribe {
	return func(hooks *hooks) {
		hooks.DoOnComplete(fn)
	}
}

// OnSubscribe sets handler for OnSubscribe.
// Also you can use DoOnSubscribe in Mono or Flux.
func OnSubscribe(fn FnOnSubscribe) OptSubscribe {
	return func(hooks *hooks) {
		hooks.DoOnSubscribe(fn)
	}
}

// OnError sets handler for OnError.
// Also you can use DoOnError in Mono or Flux.
func OnError(fn FnOnError) OptSubscribe {
	return func(hooks *hooks) {
		hooks.DoOnError(fn)
	}
}
