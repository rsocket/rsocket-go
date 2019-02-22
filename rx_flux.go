package rsocket

import (
	"context"
	"sync"
)

type implFlux struct {
	locker   *sync.Mutex
	source   chan Payload
	creation func(ctx context.Context, emitter Emitter)
	c, s     Scheduler
	handlers RxFunc
	sig      SignalType
	err      error
}

func (p *implFlux) DoOnError(onError OnError) Flux {
	p.locker.Lock()
	p.handlers.RegisterError(onError)
	p.locker.Unlock()
	return p
}

func (p *implFlux) DoOnComplete(fn OnComplete) Flux {
	p.locker.Lock()
	p.handlers.RegisterComplete(fn)
	p.locker.Unlock()
	return p
}

func (p *implFlux) DoOnCancel(onCancel OnCancel) Flux {
	p.locker.Lock()
	p.handlers.RegisterCancel(onCancel)
	p.locker.Unlock()
	return p
}

func (p *implFlux) DoOnNext(onNext Consumer) Flux {
	p.locker.Lock()
	p.handlers.RegisterNext(onNext)
	p.locker.Unlock()
	return p
}

func (p *implFlux) DoFinally(fn OnFinally) Flux {
	p.locker.Lock()
	p.handlers.RegisterFinally(fn)
	p.locker.Unlock()
	return p
}

func (p *implFlux) Error(e error) {
	p.locker.Lock()
	defer p.locker.Unlock()
	if p.sig != SigReady {
		return
	}
	p.sig = SigError
	p.err = e
	close(p.source)
}

func (p *implFlux) Dispose() {
	p.locker.Lock()
	defer p.locker.Unlock()
	if p.sig != SigReady {
		return
	}
	p.sig = SigCancel
	close(p.source)
}

func (p *implFlux) Next(item Payload) {
	defer func() {
		if e := recover(); e != nil {
			logger.Warnf("submit next item failed: %s\n", e)
		}
	}()
	p.source <- item
}

func (p *implFlux) Complete() {
	p.locker.Lock()
	defer p.locker.Unlock()
	if p.sig != SigReady {
		return
	}
	p.sig = SigSuccess
	close(p.source)
}

func (p *implFlux) SubscribeOn(scheduler Scheduler) indexedPublisher {
	p.s = scheduler
	return p
}

func (p *implFlux) Subscribe(ctx context.Context, consumer Consumer) Disposable {
	return p.subscribeIndexed(ctx, func(ctx context.Context, item Payload, i int) {
		consumer(ctx, item)
	})
}

func (p *implFlux) subscribeIndexed(ctx context.Context, consumer consumerIndexed) Disposable {
	ctx, cancel := context.WithCancel(ctx)
	if p.creation != nil {
		p.c.Do(ctx, func(ctx context.Context) {
			select {
			case <-ctx.Done():
				logger.Debugf("flux creation contex done: %s\n", ctx.Err())
				return
			default:
				p.creation(ctx, p)
			}
		})
	}
	p.s.Do(ctx, func(ctx context.Context) {
		defer func() {
			p.handlers.DoFinally(ctx)
		}()
		var stop bool
		idx := 0
		for {
			if stop {
				break
			}
			select {
			case it, ok := <-p.source:
				if !ok {
					stop = true
					break
				}
				p.handlers.DoNextOrSuccess(ctx, it)
				consumer(ctx, it, idx)
				p.handlers.DoAfterConsumer(ctx, it)
				idx++
			}
		}
		switch p.sig {
		case SigCancel:
			p.handlers.DoCancel(ctx)
			cancel()
		case SigSuccess:
			p.handlers.DoComplete(ctx)
		case SigError:
			p.handlers.DoError(ctx, p.err)
		}
	})
	return p
}

func (p *implFlux) onAfterSubscribe(fn Consumer) Flux {
	p.locker.Lock()
	p.handlers.RegisterAfterConsumer(fn)
	p.locker.Unlock()
	return p
}

func newImplFlux() *implFlux {
	return &implFlux{
		source:   make(chan Payload, 1),
		s:        ImmediateScheduler(),
		c:        ElasticScheduler(),
		locker:   &sync.Mutex{},
		sig:      SigReady,
		handlers: newRxFuncStore(),
	}
}

func NewFlux(create func(ctx context.Context, emitter Emitter)) Flux {
	if create == nil {
		panic(ErrInvalidEmitter)
	}
	f := newImplFlux()
	f.creation = create
	return f
}
