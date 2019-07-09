package rx

import "github.com/rsocket/rsocket-go/payload"

type FnOnComplete = func()
type FnOnNext = func(input payload.Payload)
type FnOnSubscribe = func(s Subscription)
type FnOnError = func(e error)
type FnOnCancel = func()
type FnFinally = func(s SignalType)
type FnPredicate = func(input payload.Payload) bool

type FnOnRequest = func(n int)
