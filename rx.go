package rsocket

import "context"

type SignalType int8

const (
	signalDefault SignalType = iota
	SignalComplete
	SignalCancel
	SignalError
	signalRequest
	signalSubscribe
	signalNext
	signalFinally
	signalNextAfter
)

type FnConsumer = func(ctx context.Context, payload Payload)
type FnOnComplete = func(ctx context.Context)
type FnOnNext = func(ctx context.Context, s Subscription, payload Payload)
type FnOnCancel = func(ctx context.Context)
type FnOnSubscribe = func(ctx context.Context, s Subscription)
type FnOnRequest = func(ctx context.Context, n int)
type FnOnError = func(ctx context.Context, err error)
type FnOnFinally = func(ctx context.Context, st SignalType)

type Disposable interface {
	Dispose()
}

type Publisher interface {
	Subscribe(ctx context.Context, ops ...OpSubscriber) Disposable
}

type Subscriber interface {
	OnSubscribe(ctx context.Context, s Subscription)
	OnNext(ctx context.Context, s Subscription, payload Payload)
	OnComplete(ctx context.Context)
	OnError(ctx context.Context, err error)
}

type Subscription interface {
	Request(n int)
	Cancel()
	n() int
}

type Processor interface {
	Publisher
	Subscriber
}

type Producer interface {
	Next(payload Payload)
	Error(err error)
	Complete()
}

type MonoProducer interface {
	Success(payload Payload)
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
