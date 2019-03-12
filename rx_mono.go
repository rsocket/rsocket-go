package rsocket

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type defaultMonoProcessor struct {
	lock         *sync.Mutex
	gen          func(MonoProducer)
	hooks        *hooks
	pubScheduler Scheduler
	subScheduler Scheduler
	r            Payload
	e            error
	sig          SignalType
	done         chan struct{}
}

func (p *defaultMonoProcessor) n() int {
	return 0
}

func (p *defaultMonoProcessor) DoAfterSuccess(fn FnConsumer) Mono {
	p.hooks.DoOnAfterNext(fn)
	return p
}

func (p *defaultMonoProcessor) Dispose() {
	p.Cancel()
}

func (p *defaultMonoProcessor) DoFinally(fn FnOnFinally) Mono {
	p.hooks.DoOnFinally(fn)
	return p
}

func (p *defaultMonoProcessor) DoOnSubscribe(fn FnOnSubscribe) Mono {
	p.hooks.DoOnSubscribe(fn)
	return p
}

func (p *defaultMonoProcessor) DoOnSuccess(fn FnOnNext) Mono {
	p.hooks.DoOnNext(fn)
	return p
}

func (p *defaultMonoProcessor) DoOnError(fn FnOnError) Mono {
	p.hooks.DoOnError(fn)
	return p
}

func (p *defaultMonoProcessor) DoOnCancel(fn FnOnCancel) Mono {
	p.hooks.DoOnCancel(fn)
	return p
}

func (p *defaultMonoProcessor) Cancel() {
	if p.sig != signalDefault {
		return
	}
	p.sig = SignalCancel
	close(p.done)
}

func (p *defaultMonoProcessor) Request(n int) {
}

func (p *defaultMonoProcessor) OnSubscribe(ctx context.Context, s Subscription) {
	p.hooks.OnSubscribe(ctx, s)
}

func (p *defaultMonoProcessor) OnNext(ctx context.Context, s Subscription, payload Payload) {
	p.hooks.OnNext(ctx, s, payload)
}

func (p *defaultMonoProcessor) OnComplete(ctx context.Context) {
	p.hooks.OnComplete(ctx)
}

func (p *defaultMonoProcessor) OnError(ctx context.Context, err error) {
	p.hooks.OnError(ctx, err)
}

func (p *defaultMonoProcessor) Success(payload Payload) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.sig == signalDefault {
		p.r = payload
		p.sig = SignalComplete
		close(p.done)
	}
}

func (p *defaultMonoProcessor) Error(err error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.sig == signalDefault {
		p.e = err
		p.sig = SignalError
		close(p.done)
	}
}

func (p *defaultMonoProcessor) SubscribeOn(s Scheduler) Mono {
	p.subScheduler = s
	return p
}

func (p *defaultMonoProcessor) PublishOn(s Scheduler) Mono {
	p.pubScheduler = s
	return p
}

func (p *defaultMonoProcessor) Subscribe(ctx context.Context, others ...OpSubscriber) Disposable {
	for _, fn := range others {
		fn(p.hooks)
	}
	if p.gen != nil {
		p.pubScheduler.Do(ctx, func(ctx context.Context) {
			defer func() {
				rec := recover()
				if rec == nil {
					return
				}
				switch v := rec.(type) {
				case error:
					p.Error(v)
				case string:
					p.Error(errors.New(v))
				default:
					p.Error(fmt.Errorf("%v", v))
				}
			}()
			p.gen(p)
		})
	}
	p.subScheduler.Do(ctx, func(ctx context.Context) {
		defer func() {
			p.hooks.OnFinally(ctx, p.sig)
		}()
		p.OnSubscribe(ctx, p)
		<-p.done
		switch p.sig {
		case SignalComplete:
			p.OnNext(ctx, p, p.r)
			p.hooks.OnAfterNext(ctx, p.r)
		case SignalError:
			p.OnError(ctx, p.e)
		case SignalCancel:
			p.hooks.OnCancel(ctx)
		}
	})
	return p
}

func NewMono(fn func(sink MonoProducer)) Mono {
	return &defaultMonoProcessor{
		lock:         &sync.Mutex{},
		gen:          fn,
		done:         make(chan struct{}),
		sig:          signalDefault,
		pubScheduler: ImmediateScheduler(),
		subScheduler: ImmediateScheduler(),
		hooks:        newHooks(),
	}
}
