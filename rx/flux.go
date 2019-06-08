package rx

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/rsocket/rsocket-go/internal/logger"
	"github.com/rsocket/rsocket-go/payload"
)

type fluxProcessor struct {
	lock         *sync.Mutex
	gen          func(context.Context, Producer)
	q            *queue
	e            error
	hooks        *hooks
	sig          SignalType
	pubScheduler Scheduler
	subScheduler Scheduler
}

func (p *fluxProcessor) N() int {
	return int(p.q.Tickets())
}

func (p *fluxProcessor) DoAfterNext(fn FnConsumer) Flux {
	p.hooks.DoOnAfterNext(fn)
	return p
}

func (p *fluxProcessor) Dispose() {
	p.Cancel()
}

func (p *fluxProcessor) IsDisposed() (ok bool) {
	p.lock.Lock()
	ok = p.sig == SignalCancel
	p.lock.Unlock()
	return
}

func (p *fluxProcessor) DoFinally(fn FnOnFinally) Flux {
	p.hooks.DoOnFinally(fn)
	return p
}

func (p *fluxProcessor) LimitRate(n int) Flux {
	// TODO: rate support
	panic("TODO: support limit rate")
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
		p.q.Request(math.MaxInt32)
	} else if n < 0 {
		p.q.Request(0)
	} else {
		p.q.Request(int32(n))
	}
}

func (p *fluxProcessor) Cancel() {
	p.lock.Lock()
	if p.sig == signalDefault {
		p.sig = SignalCancel
		_ = p.q.Close()
	}
	p.lock.Unlock()
}

func (p *fluxProcessor) Next(elem payload.Payload) (err error) {
	p.lock.Lock()
	if p.sig == signalDefault {
		err = p.q.Push(elem)
	} else {
		logger.Errorf("emit next failed: %s\n", errWrongSignal.Error())
		err = errWrongSignal
	}
	p.lock.Unlock()
	return
}

func (p *fluxProcessor) Error(e error) {
	p.lock.Lock()
	if p.sig == signalDefault {
		p.e = e
		p.sig = SignalError
		_ = p.q.Close()
	}
	p.lock.Unlock()
}

func (p *fluxProcessor) Complete() {
	p.lock.Lock()
	if p.sig == signalDefault {
		p.sig = SignalComplete
		_ = p.q.Close()
	}
	p.lock.Unlock()
}

func (p *fluxProcessor) Subscribe(ctx context.Context, ops ...OptSubscribe) Disposable {
	for _, it := range ops {
		it(p.hooks)
	}
	ctx, cancel := context.WithCancel(ctx)
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
	p.q.HandleRequest(func(n int32) {
		p.hooks.OnRequest(ctx, int(n))
	})
	p.subScheduler.Do(ctx, func(ctx context.Context) {
		defer func() {
			p.hooks.OnFinally(ctx, p.sig)
			returnHooks(p.hooks)
			p.hooks = nil
		}()
		p.OnSubscribe(ctx, p)
		for {
			v, ok := p.q.Poll(ctx)
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
			cancel()
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
		hooks:        borrowHooks(),
		gen:          fn,
		q:            newQueue(defaultQueueSize, RequestInfinite),
		pubScheduler: ElasticScheduler(),
		subScheduler: ImmediateScheduler(),
	}
}
