package flux

import (
	"context"

	"github.com/jjeffcaii/reactor-go/flux"
	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

// FnSwitchOnFirst is an alias of Func for DoSwitchOnFirst.
type FnSwitchOnFirst = func(s Signal, f Flux) Flux

// Sink represent a wrapper API around an actual downstream Subscriber for emitting nothing, a single value or an error (mutually exclusive).
type Sink interface {
	// Next emits next single value
	Next(v payload.Payload)
	// Complete complete current subscription.
	Complete()
	// Error emits a error and complete current subscription.
	Error(e error)
}

// Flux represents represents a reactive sequence of 0..N items.
type Flux interface {
	rx.Publisher
	// Take take only the first N values from this Flux, if available.
	Take(n int) Flux
	// Filter evaluate each source value against the given Predicate.
	// If the predicate test succeeds, the value is emitted.
	// If the predicate test fails, the value is ignored and a request of 1 is made upstream.
	Filter(rx.FnPredicate) Flux
	// DoOnError add behavior triggered when the Flux completes with an error.
	DoOnError(rx.FnOnError) Flux
	// DoOnNext add behavior triggered when the Flux emits an item.
	DoOnNext(rx.FnOnNext) Flux
	// DoOnComplete add behavior triggered when the Flux completes successfully.
	DoOnComplete(rx.FnOnComplete) Flux
	// DoFinally add behavior triggered after the Flux terminates for any reason, including cancellation.
	DoFinally(rx.FnFinally) Flux
	// DoOnRequest add behavior triggered after this Flux receives any request.
	DoOnRequest(rx.FnOnRequest) Flux
	// DoOnSubscribe add behavior triggered when the Flux is done being subscribed.
	DoOnSubscribe(rx.FnOnSubscribe) Flux
	// Map transform the items emitted by this Flux by applying a synchronous function to each item.
	Map(rx.FnTransform) Flux
	// SwitchOnFirst transform the current Flux once it emits its first element, making a conditional transformation possible.
	SwitchOnFirst(FnSwitchOnFirst) Flux
	// SubscribeOn run subscribe, onSubscribe and request on a specified scheduler.
	SubscribeOn(scheduler.Scheduler) Flux
	// SubscribeWithChan subscribe to this Flux and puts items/error into a chan.
	SubscribeWithChan(ctx context.Context, values chan<- payload.Payload, err chan<- error)
	// Raw returns low-level reactor.Flux which defined in reactor-go library.
	Raw() flux.Flux
	// BlockFirst subscribe to this Flux and block indefinitely until the upstream signals its first value or completes.
	// Returns that value, error if Flux completes error, or nil if the Flux completes empty.
	BlockFirst(context.Context) (payload.Payload, error)
	// BlockLast subscribe to this Flux and block indefinitely until the upstream signals its last value or completes.
	// Returns that value, error if Flux completes error, or nil if the Flux completes empty.
	BlockLast(context.Context) (payload.Payload, error)
	// ToChan subscribe to this Flux and puts items into a chan.
	// It also puts errors into another chan.
	ToChan(ctx context.Context, cap int) (c <-chan payload.Payload, e <-chan error)
	// BlockSlice subscribe to this Flux and convert to payload slice.
	BlockSlice(context.Context) ([]payload.Payload, error)
}

// Processor represent a base processor that exposes Flux API for Processor.
// See https://github.com/reactive-streams/reactive-streams-jvm/blob/v1.0.3/README.md#4processor-code.
type Processor interface {
	Sink
	Flux
}

// Signal represents a Reactive Stream signal.
type Signal interface {
	// Value returns value of Signal.
	Value() (payload.Payload, bool)
	// Type returns type of Signal.
	Type() rx.SignalType
}
