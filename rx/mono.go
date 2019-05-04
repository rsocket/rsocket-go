package rx

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/rsocket/rsocket-go/payload"
)

type defaultMonoProcessor struct {
	lock         *sync.Mutex
	gen          func(context.Context, MonoProducer)
	hooks        *hooks
	pubScheduler Scheduler
	subScheduler Scheduler
	r            payload.Payload
	e            error
	sig          SignalType
	done         chan struct{}
}

func (p *defaultMonoProcessor) N() int {
	return 0
}

func (p *defaultMonoProcessor) DoAfterSuccess(fn FnConsumer) Mono {
	p.hooks.DoOnAfterNext(fn)
	return p
}

func (p *defaultMonoProcessor) Dispose() {
	p.Cancel()
}

func (p *defaultMonoProcessor) IsDisposed() (ok bool) {
	p.lock.Lock()
	ok = p.sig == SignalCancel
	p.lock.Unlock()
	return
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
	p.lock.Lock()
	if p.sig != signalDefault {
		p.sig = SignalCancel
		close(p.done)
	}
	p.lock.Unlock()
}

func (p *defaultMonoProcessor) Request(n int) {
}

func (p *defaultMonoProcessor) OnSubscribe(ctx context.Context, s Subscription) {
	p.hooks.OnSubscribe(ctx, s)
}

func (p *defaultMonoProcessor) OnNext(ctx context.Context, s Subscription, elem payload.Payload) {
	p.hooks.OnNext(ctx, s, elem)
}

func (p *defaultMonoProcessor) OnComplete(ctx context.Context) {
	p.hooks.OnComplete(ctx)
}

func (p *defaultMonoProcessor) OnError(ctx context.Context, err error) {
	p.hooks.OnError(ctx, err)
}

func (p *defaultMonoProcessor) Success(elem payload.Payload) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.sig != signalDefault {
		return errWrongSignal
	}
	p.r = elem
	p.sig = SignalComplete
	close(p.done)
	return nil
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

func (p *defaultMonoProcessor) Subscribe(ctx context.Context, others ...OptSubscribe) Disposable {
	for _, fn := range others {
		fn(p.hooks)
	}
	ctx, cancel := context.WithCancel(ctx)
	if p.gen != nil {
		sche := p.pubScheduler
		if sche == nil {
			sche = p.subScheduler
		}
		sche.Do(ctx, func(ctx context.Context) {
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
			p.gen(ctx, p)
		})
	}
	p.subScheduler.Do(ctx, func(ctx context.Context) {
		defer func() {
			p.hooks.OnFinally(ctx, p.sig)
			returnHooks(p.hooks)
			p.hooks = nil
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
			cancel()
			p.hooks.OnCancel(ctx)
		}
	})
	return p
}

// NewMono returns a new Mono.
func NewMono(fn func(ctx context.Context, sink MonoProducer)) Mono {
	return &defaultMonoProcessor{
		lock:         &sync.Mutex{},
		gen:          fn,
		done:         make(chan struct{}),
		sig:          signalDefault,
		subScheduler: ImmediateScheduler(),
		hooks:        borrowHooks(),
	}
}
