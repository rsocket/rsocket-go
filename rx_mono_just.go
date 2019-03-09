package rsocket

import (
	"context"
)

type justMonoProcessor struct {
	item         Payload
	hooks        *Hooks
	subScheduler Scheduler
}

func (p *justMonoProcessor) n() int {
	return 0
}

func (p *justMonoProcessor) DoAfterSuccess(fn FnConsumer) Mono {
	p.hooks.DoOnAfterNext(fn)
	return p
}

func (p *justMonoProcessor) Dispose() {
}

func (p *justMonoProcessor) Cancel() {
}

func (p *justMonoProcessor) Request(n int) {
}

func (p *justMonoProcessor) OnSubscribe(ctx context.Context, s Subscription) {
	p.hooks.OnSubscribe(ctx, s)
}

func (p *justMonoProcessor) OnNext(ctx context.Context, s Subscription, payload Payload) {
	p.hooks.OnNext(ctx, s, payload)
}

func (p *justMonoProcessor) OnComplete(ctx context.Context) {
	p.hooks.OnComplete(ctx)
}

func (p *justMonoProcessor) OnError(ctx context.Context, err error) {
	p.hooks.OnError(ctx, err)
}

func (p *justMonoProcessor) DoOnSubscribe(fn FnOnSubscribe) Mono {
	p.hooks.DoOnSubscribe(fn)
	return p
}

func (p *justMonoProcessor) DoOnSuccess(fn FnOnNext) Mono {
	p.hooks.DoOnNext(fn)
	return p
}

func (p *justMonoProcessor) DoOnError(fn FnOnError) Mono {
	p.hooks.DoOnError(fn)
	return p
}

func (p *justMonoProcessor) DoOnCancel(fn FnOnCancel) Mono {
	p.hooks.DoOnCancel(fn)
	return p
}

func (p *justMonoProcessor) DoFinally(fn FnOnFinally) Mono {
	p.hooks.DoOnFinally(fn)
	return p
}

func (p *justMonoProcessor) SubscribeOn(s Scheduler) Mono {
	p.subScheduler = s
	return p
}

func (p *justMonoProcessor) PublishOn(s Scheduler) Mono {
	return p
}

func (p *justMonoProcessor) Subscribe(ctx context.Context, ops ...OpSubscriber) Disposable {
	for _, it := range ops {
		it(p.hooks)
	}
	p.subScheduler.Do(ctx, func(ctx context.Context) {
		defer p.hooks.OnFinally(ctx, SignalComplete)
		p.OnSubscribe(ctx, p)
		p.OnNext(ctx, p, p.item)
		p.hooks.OnAfterNext(ctx, p.item)
	})
	return p
}

func JustMono(item Payload) Mono {
	return &justMonoProcessor{
		subScheduler: ImmediateScheduler(),
		hooks:        NewHooks(),
		item:         item,
	}
}
