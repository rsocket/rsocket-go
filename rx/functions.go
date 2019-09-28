package rx

import "github.com/rsocket/rsocket-go/payload"

type (
	// FnOnComplete is alias of function for signal when no more elements are available
	FnOnComplete = func()
	// FnOnNext is alias of function for signal when next element arrived.
	FnOnNext = func(input payload.Payload)
	// FnOnSubscribe is alias of function for signal when subscribe begin.
	FnOnSubscribe = func(s Subscription)
	// FnOnError is alias of function for signal when an error occured.
	FnOnError = func(e error)
	// FnOnCancel is alias of function for signal when subscription canceled.
	FnOnCancel = func()
	// FnFinally is alias of function for signal when all things done.
	FnFinally = func(s SignalType)
	// FnPredicate is alias of function for filter operations.
	FnPredicate = func(input payload.Payload) bool
	// FnOnRequest is alias of function for signal when requesting next element.
	FnOnRequest = func(n int)
)
