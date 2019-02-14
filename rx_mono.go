package rsocket

import (
	"context"
	"sync"
)

type implMono struct {
	ob func(emitter MonoEmitter)

	once *sync.Once
	done chan struct{}

	success Payload

	schedulerC Scheduler
	schedulerS Scheduler

	onFinally []OnFinally
	handlers  []Consumer
	afterSub  Consumer
}

func (p *implMono) Error(e error) {
	panic("implement me")
}

func (p *implMono) Dispose() {
	panic("implement me")
}

func (p *implMono) setAfterSub(consumer Consumer) {
	p.afterSub = consumer
}

func (p *implMono) DoOnNext(onNext Consumer) Mono {
	p.handlers = append(p.handlers, func(ctx context.Context, item Payload) {
		if item != nil {
			onNext(ctx, item)
		}
	})
	return p
}

func (p *implMono) Success(payload Payload) {
	p.once.Do(func() {
		p.success = payload
		close(p.done)
	})
}

func (p *implMono) Subscribe(ctx context.Context, consumer Consumer) Disposable {
	if p.ob != nil {
		p.schedulerC.Do(ctx, func(ctx context.Context) {
			p.ob(p)
		})
	}
	p.schedulerS.Do(ctx, func(ctx context.Context) {
		<-p.done
		for _, fn := range p.handlers {
			fn(ctx, p.success)
		}
		if p.success != nil {
			consumer(ctx, p.success)
			if p.afterSub != nil {
				p.afterSub(ctx, p.success)
			}
		}
		defer func() {
			for i, l := 0, len(p.onFinally); i < l; i++ {
				p.onFinally[l-i-1](ctx)
			}
		}()
	})
	return p
}

func (p *implMono) DoOnSuccess(onSuccess Consumer) Mono {
	p.handlers = append(p.handlers, onSuccess)
	return p
}

func (p *implMono) DoFinally(fn OnFinally) Mono {
	p.onFinally = append(p.onFinally, fn)
	return p
}

func (p *implMono) SubscribeOn(scheduler Scheduler) Publisher {
	p.schedulerS = scheduler
	return p
}
