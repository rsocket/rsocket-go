package rsocket

import (
	"context"
	"sync"
)

type implMonoSingle struct {
	value    Payload
	locker   *sync.Mutex
	handlers RxFunc
	ss       Scheduler
}

func (p *implMonoSingle) Dispose() {
}

func (p *implMonoSingle) DoOnCancel(onCancel OnCancel) Mono {
	return p
}

func (p *implMonoSingle) DoOnNext(onNext Consumer) Mono {
	p.locker.Lock()
	p.handlers.RegisterSuccess(func(ctx context.Context, item Payload) {
		if item != nil {
			onNext(ctx, item)
		}
	})
	p.locker.Unlock()
	return p
}

func (p *implMonoSingle) DoOnSuccess(onSuccess Consumer) Mono {
	p.locker.Lock()
	p.handlers.RegisterSuccess(onSuccess)
	p.locker.Unlock()
	return p
}

func (p *implMonoSingle) DoOnError(onError OnError) Mono {
	return p
}

func (p *implMonoSingle) DoFinally(fn OnFinally) Mono {
	p.locker.Lock()
	p.handlers.RegisterFinally(fn)
	p.locker.Unlock()
	return p
}

func (p *implMonoSingle) SubscribeOn(scheduler Scheduler) Publisher {
	p.ss = scheduler
	return p
}

func (p *implMonoSingle) onAfterSubscribe(fn Consumer) Mono {
	p.locker.Lock()
	p.handlers.RegisterAfterConsumer(fn)
	p.locker.Unlock()
	return p
}

func (p *implMonoSingle) Subscribe(ctx context.Context, consumer Consumer) Disposable {
	p.ss.Do(ctx, func(ctx context.Context) {
		select {
		case <-ctx.Done():
			defer p.handlers.DoFinally(ctx, SigCancel)
			return
		default:
			defer p.handlers.DoFinally(ctx, SigSuccess)
			p.handlers.DoNextOrSuccess(ctx, p.value)
			consumer(ctx, p.value)
			p.handlers.DoAfterConsumer(ctx, p.value)
		}
	})
	return p
}
