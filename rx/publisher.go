package rx

import "context"

type RawPublisher interface {
	SubscribeWith(ctx context.Context, s Subscriber)
}

// Publisher is a provider of a potentially unbounded number of sequenced elements, \
// publishing them according to the demand received from its Subscriber(s).
type Publisher interface {
	RawPublisher
	// Subscribe subscribe elements from a publisher, returns a Disposable.
	// You can add some custome options.
	// Using `OnSubscribe`, `OnNext`, `OnComplete` and `OnError` as handler wrapper.
	Subscribe(ctx context.Context, options ...SubscriberOption)
}
