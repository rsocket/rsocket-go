package rsocket

import (
	"sync"
)

type Consumer = func(item Payload)
type OnFinally = func()

type Publisher interface {
	Subscribe(consumer Consumer)
}

type Flux interface {
	Publisher
	DoOnNext(onNext Consumer) Flux
	DoFinally(fn func()) Flux
	SubscribeOn(scheduler Scheduler) Publisher
}

type Emitter interface {
	Next(item Payload)
	Complete()
}

type implFlux struct {
	source        chan Payload
	creation      func(emitter Emitter)
	closeOnce     *sync.Once
	subScheduler  Scheduler
	doFinallies   []OnFinally
	doOnNexts     []Consumer
	afterConsumer Consumer
}

func (p *implFlux) DoOnNext(onNext Consumer) Flux {
	if onNext != nil {
		p.doOnNexts = append(p.doOnNexts, onNext)
	}
	return p
}

func (p *implFlux) setAfterConsumer(consumer Consumer) {
	p.afterConsumer = consumer
}

func (p *implFlux) DoFinally(fn OnFinally) Flux {
	if fn != nil {
		p.doFinallies = append(p.doFinallies, fn)
	}
	return p
}

func (p *implFlux) SubscribeOn(scheduler Scheduler) Publisher {
	p.subScheduler = scheduler
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

func (p *implFlux) Subscribe(consumer Consumer) {
	if p.creation != nil {
		go p.creation(p)
	}
	p.subScheduler.Do(func() {
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
					onNext(it)
				}
				consumer(it)
				if p.afterConsumer != nil {
					p.afterConsumer(it)
				}
			}
		}
		for i, l := 0, len(p.doFinallies); i < l; i++ {
			p.doFinallies[l-i-1]()
		}
	})
}

func newImplFlux() *implFlux {
	return &implFlux{
		source:       make(chan Payload),
		doOnNexts:    make([]Consumer, 0),
		closeOnce:    &sync.Once{},
		subScheduler: ImmediateScheduler(),
	}
}

func NewFlux(h func(emitter Emitter)) Flux {
	if h == nil {
		panic(ErrInvalidEmitter)
	}
	f := newImplFlux()
	f.creation = h
	return f
}
