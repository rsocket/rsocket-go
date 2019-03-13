package rsocket

import "context"

// SignalType is the signal of reactive events like `OnNext`, `OnComplete`, `OnCancel` and `OnError`.
type SignalType int8

const (
	signalDefault SignalType = iota
	// SignalComplete indicated that subscriber was completed.
	SignalComplete
	// SignalCancel indicates that subscriber was cancelled.
	SignalCancel
	// SignalError indicates that subscriber has some faults.
	SignalError
	signalRequest
	signalSubscribe
	signalNext
	signalFinally
	signalNextAfter
)

// FnConsumer is alias of consumer function.
type FnConsumer = func(ctx context.Context, payload Payload)

// FnOnComplete is alias of `OnComplete` handler.
type FnOnComplete = func(ctx context.Context)

// FnOnNext is alias of `OnNext` handler.
type FnOnNext = func(ctx context.Context, s Subscription, payload Payload)

// FnOnCancel is alias of `OnCancel` handler.
type FnOnCancel = func(ctx context.Context)

// FnOnSubscribe is alias of `OnSubscribe` handler.
type FnOnSubscribe = func(ctx context.Context, s Subscription)

// FnOnRequest is alias of `OnRequest` handler.
type FnOnRequest = func(ctx context.Context, n int)

// FnOnError is alias of `OnError` handler.
type FnOnError = func(ctx context.Context, err error)

// FnOnFinally is alias of `OnFinally` handler.
type FnOnFinally = func(ctx context.Context, st SignalType)

// Disposable is a disposable resource.
type Disposable interface {
	// Dispose dispose current resource.
	Dispose()
}

// Publisher is a provider of a potentially unbounded number of sequenced elements, \
// publishing them according to the demand received from its Subscriber(s).
type Publisher interface {
	// Subscribe subscribe elements from a publisher, returns a Disposable.
	// You can add some custome options.
	// Using `OnSubscribe`, `OnNext`, `OnComplete` and `OnError` as handler wrapper.
	Subscribe(ctx context.Context, ops ...OpSubscriber) Disposable
}

// Subscriber consume elements from a Publisher and handle events.
type Subscriber interface {
	// OnSubscribe handle event when subscribe begin.
	OnSubscribe(ctx context.Context, s Subscription)
	// OnNext handle event when a new element produced.
	OnNext(ctx context.Context, s Subscription, payload Payload)
	// OnComplete handle event when subscribe finish.
	OnComplete(ctx context.Context)
	// OnError handle event when an error occurred。
	OnError(ctx context.Context, err error)
}

// Subscription means a Subscrber's subscription.
type Subscription interface {
	// Request pull next n elements. (It was used for FlowControl)
	// When you call it, subscriber will emit `OnRequest` event and you can use `DoOnRequest` catch it.
	Request(n int)
	// Cancel cancel the current subscriber.
	// Subscribers will emit `OnCancel` event and you can use `DoOnCancel` catch it.
	Cancel()

	n() int
}

// Processor process publisher and subscriber.
type Processor interface {
	Publisher
	Subscriber
}

// Producer produce elements as you wish.
type Producer interface {
	// Next append next element.
	Next(payload Payload)
	// Error means some bad things happened.
	Error(err error)
	// Complete means production completed.
	Complete()
}

// MonoProducer likes Producer, but it produce single element.
type MonoProducer interface {
	// Success append payload.
	Success(payload Payload)
	// Error means some bad things happened.
	Error(err error)
}

type Mono interface {
	Publisher
	DoAfterSuccess(fn FnConsumer) Mono
	DoOnSubscribe(fn FnOnSubscribe) Mono
	DoOnSuccess(fn FnOnNext) Mono
	DoOnError(fn FnOnError) Mono
	DoOnCancel(fn FnOnCancel) Mono
	DoFinally(fn FnOnFinally) Mono
	SubscribeOn(s Scheduler) Mono
	PublishOn(s Scheduler) Mono
}

type Flux interface {
	Publisher
	LimitRate(n int) Flux
	DoOnRequest(fn FnOnRequest) Flux
	DoOnSubscribe(fn FnOnSubscribe) Flux
	DoOnNext(fn FnOnNext) Flux
	DoAfterNext(fn FnConsumer) Flux
	DoOnComplete(fn FnOnComplete) Flux
	DoOnError(fn FnOnError) Flux
	DoOnCancel(fn FnOnCancel) Flux
	DoFinally(fn FnOnFinally) Flux
	SubscribeOn(s Scheduler) Flux
	PublishOn(s Scheduler) Flux
}

type OpSubscriber func(*hooks)

func OnNext(fn FnOnNext) OpSubscriber {
	return func(hooks *hooks) {
		hooks.DoOnNext(fn)
	}
}

func OnComplete(fn FnOnComplete) OpSubscriber {
	return func(hooks *hooks) {
		hooks.DoOnComplete(fn)
	}
}

func OnSubscribe(fn FnOnSubscribe) OpSubscriber {
	return func(hooks *hooks) {
		hooks.DoOnSubscribe(fn)
	}
}

func OnError(fn FnOnError) OpSubscriber {
	return func(hooks *hooks) {
		hooks.DoOnError(fn)
	}
}

type hooks struct {
	m map[SignalType][]interface{}
}

func (p *hooks) OnCancel(ctx context.Context) {
	found, ok := p.m[SignalCancel]
	if !ok {
		return
	}
	for _, fn := range found {
		fn.(FnOnCancel)(ctx)
	}
}

func (p *hooks) OnRequest(ctx context.Context, n int) {
	found, ok := p.m[signalRequest]
	if !ok {
		return
	}
	for _, fn := range found {
		fn.(FnOnRequest)(ctx, n)
	}
}

func (p *hooks) OnSubscribe(ctx context.Context, s Subscription) {
	found, ok := p.m[signalSubscribe]
	if !ok {
		return
	}
	for _, fn := range found {
		fn.(FnOnSubscribe)(ctx, s)
	}
}

func (p *hooks) OnNext(ctx context.Context, s Subscription, v Payload) {
	found, ok := p.m[signalNext]
	if !ok {
		return
	}
	for _, fn := range found {
		fn.(FnOnNext)(ctx, s, v)
	}
}

func (p *hooks) OnComplete(ctx context.Context) {
	found, ok := p.m[SignalComplete]
	if !ok {
		return
	}
	for _, fn := range found {
		fn.(FnOnComplete)(ctx)
	}
}

func (p *hooks) OnError(ctx context.Context, err error) {
	found, ok := p.m[SignalError]
	if !ok {
		return
	}
	for _, fn := range found {
		fn.(FnOnError)(ctx, err)
	}
}

func (p *hooks) OnFinally(ctx context.Context, sig SignalType) {
	found, ok := p.m[signalFinally]
	if !ok {
		return
	}
	for i, l := 0, len(found); i < l; i++ {
		found[l-i-1].(FnOnFinally)(ctx, sig)
	}
}

func (p *hooks) DoOnAfterNext(fn FnConsumer) {
	p.register(signalNextAfter, fn)
}

func (p *hooks) OnAfterNext(ctx context.Context, payload Payload) {
	found, ok := p.m[signalNextAfter]
	if !ok {
		return
	}
	for _, fn := range found {
		fn.(FnConsumer)(ctx, payload)
	}
}

func (p *hooks) DoOnError(fn FnOnError) {
	p.register(SignalError, fn)
}

func (p *hooks) DoOnNext(fn FnOnNext) {
	p.register(signalNext, fn)
}

func (p *hooks) DoOnRequest(fn FnOnRequest) {
	p.register(signalRequest, fn)
}

func (p *hooks) DoOnComplete(fn FnOnComplete) {
	p.register(SignalComplete, fn)
}

func (p *hooks) DoOnCancel(fn FnOnCancel) {
	p.register(SignalCancel, fn)
}

func (p *hooks) DoOnSubscribe(fn FnOnSubscribe) {
	p.register(signalSubscribe, fn)
}
func (p *hooks) DoOnFinally(fn FnOnFinally) {
	p.register(signalFinally, fn)
}

func (p *hooks) register(sig SignalType, fn interface{}) {
	p.m[sig] = append(p.m[sig], fn)
}

func newHooks() *hooks {
	return &hooks{
		m: make(map[SignalType][]interface{}),
	}
}
