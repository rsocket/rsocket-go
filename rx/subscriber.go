package rx

import (
	"context"

	"github.com/jjeffcaii/reactor-go"
	"github.com/rsocket/rsocket-go/payload"
)

var (
	// EmptySubscriber is a blank Subscriber.
	EmptySubscriber Subscriber = &subscriber{}
	// EmptyRawSubscriber is a blank native Subscriber in reactor-go.
	EmptyRawSubscriber = reactor.NewSubscriber(reactor.OnNext(func(v reactor.Any) error {
		return nil
	}))
)

// Subscription represents a one-to-one lifecycle of a Subscriber subscribing to a Publisher.
type Subscription = reactor.Subscription

// Subscriber will receive call to OnSubscribe(Subscription) once after passing an instance of Subscriber to Publisher#SubscribeWith
type Subscriber interface {
	// OnNext represents data notification sent by the Publisher in response to requests to Subscription#Request.
	OnNext(payload payload.Payload)
	// OnError represents failed terminal state.
	OnError(error)
	// OnComplete represents successful terminal state.
	OnComplete()
	// OnSubscribe invoked after Publisher subscribed.
	// No data will start flowing until Subscription#Request is invoked.
	OnSubscribe(context.Context, Subscription)
}

type subscriberFacade struct {
	Subscriber
}

type subscriber struct {
	fnOnSubscribe FnOnSubscribe
	fnOnNext      FnOnNext
	fnOnComplete  FnOnComplete
	fnOnError     FnOnError
}

func NewSubscriberFacade(s Subscriber) reactor.Subscriber {
	return subscriberFacade{
		Subscriber: s,
	}
}

func (s subscriberFacade) OnNext(any reactor.Any) {
	s.Subscriber.OnNext(any.(payload.Payload))
}

func (s *subscriber) OnNext(payload payload.Payload) {
	if s == nil || s.fnOnNext == nil {
		return
	}
	if err := s.fnOnNext(payload); err != nil {
		s.OnError(err)
	}
}

func (s *subscriber) OnError(err error) {
	if s != nil && s.fnOnError != nil {
		s.fnOnError(err)
	}
}

func (s *subscriber) OnComplete() {
	if s != nil && s.fnOnComplete != nil {
		s.fnOnComplete()
	}
}

func (s *subscriber) OnSubscribe(ctx context.Context, su Subscription) {
	if s != nil && s.fnOnSubscribe != nil {
		s.fnOnSubscribe(ctx, su)
	} else {
		su.Request(RequestMax)
	}
}

// SubscriberOption is option of subscriber.
// You can call OnNext, OnComplete, OnError or OnSubscribe.
type SubscriberOption func(*subscriber)

// OnNext returns s SubscriberOption handling Next event.
func OnNext(onNext FnOnNext) SubscriberOption {
	return func(s *subscriber) {
		s.fnOnNext = onNext
	}
}

// OnComplete returns s SubscriberOption handling Complete event.
func OnComplete(onComplete FnOnComplete) SubscriberOption {
	return func(s *subscriber) {
		s.fnOnComplete = onComplete
	}
}

// OnError returns s SubscriberOption handling Error event.
func OnError(onError FnOnError) SubscriberOption {
	return func(i *subscriber) {
		i.fnOnError = onError
	}
}

// OnSubscribe returns s SubscriberOption handling Subscribe event.
func OnSubscribe(onSubscribe FnOnSubscribe) SubscriberOption {
	return func(i *subscriber) {
		i.fnOnSubscribe = onSubscribe
	}
}

// NewSubscriber create a new Subscriber with custom options.
func NewSubscriber(opts ...SubscriberOption) Subscriber {
	if len(opts) < 1 {
		return EmptySubscriber
	}
	s := &subscriber{}
	for _, opt := range opts {
		opt(s)
	}
	return s
}
