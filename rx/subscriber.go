package rx

import (
	"github.com/jjeffcaii/reactor-go"
	"github.com/rsocket/rsocket-go/payload"
)

var EmptySubscriber Subscriber
var EmptyRawSubscriber rs.Subscriber

func init() {
	EmptySubscriber = &subscriber{}
	EmptyRawSubscriber = rs.NewSubscriber(rs.OnNext(func(v interface{}) {
	}))
}

type Subscription rs.Subscription

type Subscriber interface {
	OnNext(payload payload.Payload)
	OnError(error)
	OnComplete()
	OnSubscribe(Subscription)
}

type subscriber struct {
	fnOnSubscribe FnOnSubscribe
	fnOnNext      FnOnNext
	fnOnComplete  FnOnComplete
	fnOnError     FnOnError
}

func (s *subscriber) OnNext(payload payload.Payload) {
	if s != nil && s.fnOnNext != nil {
		s.fnOnNext(payload)
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

func (s *subscriber) OnSubscribe(su Subscription) {
	if s != nil && s.fnOnSubscribe != nil {
		s.fnOnSubscribe(su)
	} else {
		su.Request(RequestMax)
	}
}

type SubscriberOption func(*subscriber)

func OnNext(onNext FnOnNext) SubscriberOption {
	return func(s *subscriber) {
		s.fnOnNext = onNext
	}
}

func OnComplete(onComplete FnOnComplete) SubscriberOption {
	return func(s *subscriber) {
		s.fnOnComplete = onComplete
	}
}

func OnError(onError FnOnError) SubscriberOption {
	return func(i *subscriber) {
		i.fnOnError = onError
	}
}

func OnSubscribe(onSubscribe FnOnSubscribe) SubscriberOption {
	return func(i *subscriber) {
		i.fnOnSubscribe = onSubscribe
	}
}

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
