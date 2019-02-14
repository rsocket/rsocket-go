package rsocket

import (
	"context"
	"sync"
)

type implFlux struct {
	source        chan Payload
	creation      func(ctx context.Context, emitter Emitter)
	closeOnce     *sync.Once
	schedulerC    Scheduler
	schedulerS    Scheduler
	doFinallies   []OnFinally
	doOnNexts     []Consumer
	afterConsumer Consumer
}

func (p *implFlux) Error(e error) {
	panic("implement me")
}

func (p *implFlux) Dispose() {
	panic("implement me")
}

func (p *implFlux) setAfterConsumer(consumer Consumer) {
	p.afterConsumer = consumer
}

func (p *implFlux) DoOnNext(onNext Consumer) Flux {
	if onNext != nil {
		p.doOnNexts = append(p.doOnNexts, onNext)
	}
	return p
}

func (p *implFlux) DoFinally(fn OnFinally) Flux {
	if fn != nil {
		p.doFinallies = append(p.doFinallies, fn)
	}
	return p
}

func (p *implFlux) SubscribeOn(scheduler Scheduler) Publisher {
	p.schedulerS = scheduler
	return p
}

func (p *implFlux) Next(item Payload) {
	p.source <- item
}

func (p *implFlux) Complete() {
	p.closeOnce.Do(func() {
		close(p.source)
	})
}

func (p *implFlux) Subscribe(ctx context.Context, consumer Consumer) Disposable {
	if p.creation != nil {
		p.schedulerC.Do(ctx, func(ctx context.Context) {
			p.creation(ctx, p)
		})
	}
	p.schedulerS.Do(ctx, func(ctx context.Context) {
		var stop bool
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
				for _, onNext := range p.doOnNexts {
					onNext(ctx, it)
				}
				consumer(ctx, it)
				if p.afterConsumer != nil {
					p.afterConsumer(ctx, it)
				}
			}
		}
		defer func() {
			for i, l := 0, len(p.doFinallies); i < l; i++ {
				p.doFinallies[l-i-1](ctx)
			}
		}()
	})
	return p
}

func newImplFlux() *implFlux {
	return &implFlux{
		source:     make(chan Payload),
		doOnNexts:  make([]Consumer, 0),
		closeOnce:  &sync.Once{},
		schedulerS: ImmediateScheduler(),
		schedulerC: ElasticScheduler(),
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
