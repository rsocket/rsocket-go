package mono

import (
	"context"
	"time"

	"github.com/jjeffcaii/reactor-go/mono"
	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

// ReleaseFunc can be used to release resources.
type ReleaseFunc func()
type Item = rx.Item
type Combinator2 = func(first, second Item) (payload.Payload, error)

// Mono is a Reactive Streams Publisher with basic rx operators that completes successfully by emitting an element, or with an error.
type Mono interface {
	rx.Publisher
	// Filter evaluate each source value against the given Predicate.
	// If the predicate test succeeds, the value is emitted.
	Filter(rx.FnPredicate) Mono
	// Map transform the item emitted by this Mono by applying a synchronous function to another.
	Map(rx.FnTransform) Mono
	// FlatMap Transform the item emitted by this Mono asynchronously, returning the value emitted by another Mono.
	FlatMap(func(payload.Payload) Mono) Mono
	// DoFinally adds behavior (side-effect) triggered after the Mono terminates for any reason, including cancellation.
	DoFinally(rx.FnFinally) Mono
	// DoOnError adds behavior (side-effect) triggered when the Mono completes with an error.
	DoOnError(rx.FnOnError) Mono
	// DoOnSuccess adds behavior (side-effect) triggered when the Mono completes with an success.
	DoOnSuccess(rx.FnOnNext) Mono
	// DoOnCancel add behavior (side-effect) triggered when the Mono is cancelled.
	DoOnCancel(rx.FnOnCancel) Mono
	// DoOnSubscribe add behavior (side-effect) triggered when the Mono is done being subscribed.
	DoOnSubscribe(rx.FnOnSubscribe) Mono
	// SubscribeOn customize a Scheduler running Subscribe, OnSubscribe and Request.
	SubscribeOn(scheduler.Scheduler) Mono
	// SubscribeWithChan subscribe to this Mono and puts item/error into channels.
	SubscribeWithChan(ctx context.Context, valueChan chan<- payload.Payload, errChan chan<- error)
	// BlockUnsafe blocks Mono and returns data and error.
	// Payload could be pooled sometimes, so make sure calling ReleaseFunc when you no longer need Payload, or it will cause leak problem.
	BlockUnsafe(context.Context) (payload.Payload, ReleaseFunc, error)
	// Block blocks Mono and returns a cloned payload.
	// It's different from BlockUnsafe, you don't need release it manually.
	Block(context.Context) (payload.Payload, error)
	// SwitchIfEmpty switch to an alternative Publisher if this Mono is completed without any data.
	SwitchIfEmpty(alternative Mono) Mono
	// SwitchIfError switch to an alternative Publisher if this Mono is end with an error.
	SwitchIfError(alternative func(error) Mono) Mono
	// SwitchValueIfError switch to an alternative Payload if this Mono is end with an error.
	SwitchValueIfError(alternative payload.Payload) Mono
	// ZipWith combines the result from this mono and another into a new Payload.
	ZipWith(alternative Mono, cmb Combinator2) Mono
	// Raw returns low-level reactor.Mono which defined in reactor-go library.
	Raw() mono.Mono
	// ToChan subscribe Mono and puts items into a chan.
	// It also puts errors into another chan.
	ToChan(ctx context.Context) (c <-chan payload.Payload, e <-chan error)
	// Timeout sets the timeout value.
	Timeout(timeout time.Duration) Mono
}

// Sink is a wrapper API around an actual downstream Subscriber for emitting nothing, a single value or an error (mutually exclusive).
type Sink interface {
	// Success emits a single value then complete current Sink.
	Success(payload.Payload)
	// Error emits an error then complete current Sink.
	Error(error)
}
