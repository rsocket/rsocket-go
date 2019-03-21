package rx

import (
	"context"
	"errors"
	"fmt"
	"github.com/rsocket/rsocket-go/payload"
	"math"
	"sync"
)

type fluxProcessor struct {
	lock         *sync.Mutex
	gen          func(context.Context, Producer)
	q            *bQueue
	e            error
	hooks        *hooks
	sig          SignalType
	pubScheduler Scheduler
	subScheduler Scheduler
}

func (p *fluxProcessor) N() int {
	return int(p.q.tickets)
}

func (p *fluxProcessor) DoAfterNext(fn FnConsumer) Flux {
	p.hooks.DoOnAfterNext(fn)
	return p
}

func (p *fluxProcessor) Dispose() {
	p.Cancel()
}

func (p *fluxProcessor) DoFinally(fn FnOnFinally) Flux {
	p.hooks.DoOnFinally(fn)
	return p
}

func (p *fluxProcessor) LimitRate(n int) Flux {
	if n < 1 {
		p.q.setRate(0, 0)
	} else if n >= math.MaxInt32 {
		p.q.setRate(math.MaxInt32, math.MaxInt32)
	} else {
		p.q.setRate(int32(n), int32(n))
	}
	return p
}

func (p *fluxProcessor) DoOnRequest(fn FnOnRequest) Flux {
	p.hooks.DoOnRequest(fn)
	return p
}

func (p *fluxProcessor) DoOnSubscribe(fn FnOnSubscribe) Flux {
	p.hooks.DoOnSubscribe(fn)
	return p
}

func (p *fluxProcessor) DoOnNext(fn FnOnNext) Flux {
	p.hooks.DoOnNext(fn)
	return p
}

func (p *fluxProcessor) DoOnComplete(fn FnOnComplete) Flux {
	p.hooks.DoOnComplete(fn)
	return p
}

func (p *fluxProcessor) DoOnError(fn FnOnError) Flux {
	p.hooks.DoOnError(fn)
	return p
}

func (p *fluxProcessor) DoOnCancel(fn FnOnCancel) Flux {
	p.hooks.DoOnCancel(fn)
	return p
}

func (p *fluxProcessor) SubscribeOn(s Scheduler) Flux {
	p.subScheduler = s
	return p
}

func (p *fluxProcessor) PublishOn(s Scheduler) Flux {
	p.pubScheduler = s
	return p
}

func (p *fluxProcessor) Request(n int) {
	if n > math.MaxInt32 {
		p.q.requestN(math.MaxInt32)
	} else if n < 0 {
		p.q.requestN(0)
	} else {
		p.q.requestN(int32(n))
	}
}

func (p *fluxProcessor) Cancel() {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.sig == signalDefault {
		p.sig = SignalCancel
		_ = p.q.Close()
	}

}

func (p *fluxProcessor) Next(elem payload.Payload) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.sig == signalDefault {
		return p.q.add(elem)
	}
	return errWrongSignal
}

func (p *fluxProcessor) Error(e error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.sig == signalDefault {
		p.e = e
		p.sig = SignalError
		_ = p.q.Close()
	}
}

func (p *fluxProcessor) Complete() {
	p.lock.Lock()
	defer p.lock.Unlock()
	if p.sig == signalDefault {
		p.sig = SignalComplete
		_ = p.q.Close()
	}
}

func (p *fluxProcessor) Subscribe(ctx context.Context, ops ...OptSubscribe) Disposable {
	for _, it := range ops {
		it(p.hooks)
	}
	if p.gen != nil {
		p.pubScheduler.Do(ctx, func(ctx context.Context) {
			defer func() {
				e := recover()
				if e == nil {
					return
				}
				switch v := e.(type) {
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
	// bind request N
	p.q.onRequestN = func(n int32) {
		p.hooks.OnRequest(ctx, int(n))
	}
	p.subScheduler.Do(ctx, func(ctx context.Context) {
		defer func() {
			p.hooks.OnFinally(ctx, p.sig)
		}()
		p.OnSubscribe(ctx, p)
		for {
			v, ok := p.q.poll(ctx)
			if !ok {
				break
			}
			p.OnNext(ctx, p, v)
			p.hooks.OnAfterNext(ctx, v)
		}
		switch p.sig {
		case SignalComplete:
			p.OnComplete(ctx)
		case SignalError:
			p.OnError(ctx, p.e)
		case SignalCancel:
			p.hooks.OnCancel(ctx)
		}
	})
	return p
}

func (p *fluxProcessor) OnSubscribe(ctx context.Context, s Subscription) {
	p.hooks.OnSubscribe(ctx, s)
}

func (p *fluxProcessor) OnNext(ctx context.Context, s Subscription, elem payload.Payload) {
	p.hooks.OnNext(ctx, s, elem)
}

func (p *fluxProcessor) OnComplete(ctx context.Context) {
	p.hooks.OnComplete(ctx)
}

func (p *fluxProcessor) OnError(ctx context.Context, err error) {
	p.hooks.OnError(ctx, err)
}

// NewFlux returns a new Flux.
func NewFlux(fn func(ctx context.Context, producer Producer)) Flux {
	return &fluxProcessor{
		lock:         &sync.Mutex{},
		hooks:        newHooks(),
		gen:          fn,
		q:            newQueue(16, math.MaxInt32, 0),
		pubScheduler: ElasticScheduler(),
		subScheduler: ImmediateScheduler(),
	}
}
