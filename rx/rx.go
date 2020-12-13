package rx

import (
	"context"

	"github.com/jjeffcaii/reactor-go"
	"github.com/jjeffcaii/reactor-go/hooks"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/payload"
)

func init() {
	hooks.OnNextDrop(func(v reactor.Any) {
		common.TryRelease(v)
	})
	hooks.OnErrorDrop(func(e error) {
		common.TryRelease(e)
	})
}

// RequestMax represents unbounded request amount.
const RequestMax = reactor.RequestInfinite

const (
	// SignalComplete indicated that subscriber was completed.
	SignalComplete = SignalType(reactor.SignalTypeComplete)
	// SignalCancel indicates that subscriber was cancelled.
	SignalCancel = SignalType(reactor.SignalTypeCancel)
	// SignalError indicates that subscriber has some faults.
	SignalError = SignalType(reactor.SignalTypeError)
)

type (
	// FnOnComplete is alias of function for signal when no more elements are available
	FnOnComplete = func()
	// FnOnNext is alias of function for signal when next element arrived.
	FnOnNext = func(input payload.Payload) error
	// FnOnSubscribe is alias of function for signal when subscribe begin.
	FnOnSubscribe = func(ctx context.Context, s Subscription)
	// FnOnError is alias of function for signal when an error occurred.
	FnOnError = func(e error)
	// FnOnCancel is alias of function for signal when subscription canceled.
	FnOnCancel = func()
	// FnFinally is alias of function for signal when all things done.
	FnFinally = func(s SignalType)
	// FnPredicate is alias of function for filter operations.
	FnPredicate = func(input payload.Payload) bool
	// FnOnRequest is alias of function for signal when requesting next element.
	FnOnRequest = func(n int)
	// FnTransform is alias of function to transform a payload to another.
	FnTransform = func(payload.Payload) (payload.Payload, error)
)

// RawPublisher represents a basic Publisher which can be subscribed by a Subscriber.
type RawPublisher interface {
	// SubscribeWith can be used to subscribe current publisher.
	SubscribeWith(ctx context.Context, s Subscriber)
}

// Publisher is a provider of a potentially unbounded number of sequenced elements, \
// publishing them according to the demand received from its Subscriber(s).
type Publisher interface {
	RawPublisher
	// Subscribe subscribe elements from a publisher, returns a Disposable.
	// You can add some custom options.
	// Using `OnSubscribe`, `OnNext`, `OnComplete` and `OnError` as handler wrapper.
	Subscribe(ctx context.Context, options ...SubscriberOption)
}

// SignalType is the signal of reactive events like `OnNext`, `OnComplete`, `OnCancel` and `OnError`.
type SignalType reactor.SignalType

func (s SignalType) String() string {
	return reactor.SignalType(s).String()
}

// Item is a kind of container which contains value or error.
type Item struct {
	V payload.Payload
	E error
}
