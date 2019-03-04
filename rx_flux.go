package rsocket

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
)

type implFlux struct {
	locker   *sync.Mutex
	source   chan Payload
	creation func(ctx context.Context, emitter FluxEmitter)
	c, s     Scheduler
	handlers RxFunc
	sig      SignalType
	err      error
	rl       *rateLimiter
}

func (p *implFlux) DoOnSubscribe(onSubscribe OnSubscribe) Flux {
	p.handlers.RegisterSubscribe(onSubscribe)
	return p
}

func (p *implFlux) limiter() (*rateLimiter, bool) {
	if p.rl == nil {
		return nil, false
	}
	return p.rl, true
}

func (p *implFlux) onExhaust(onExhaust OnExhaust) Flux {
	p.locker.Lock()
	p.handlers.RegisterDrain(onExhaust)
	p.locker.Unlock()
	return p
}

func (p *implFlux) resetLimit(n uint32) {
	p.locker.Lock()
	defer p.locker.Unlock()
	if p.rl == nil {
		return
	}
	p.rl.reset(n)
}

func (p *implFlux) LimitRate(n uint32) Flux {
	p.rl = newRateLimiter(n)
	return p
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
	p.release()
}

func (p *implFlux) Dispose() {
	p.locker.Lock()
	defer p.locker.Unlock()
	if p.sig != SigReady {
		return
	}
	p.sig = SigCancel
	p.release()
}

func (p *implFlux) Next(item Payload) {
	defer func() {
		if e := recover(); e != nil {
			item.Release()
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
	p.release()
}

func (p *implFlux) SubscribeOn(scheduler Scheduler) Publisher {
	p.s = scheduler
	return p
}

func (p *implFlux) Subscribe(ctx context.Context, consumer Consumer) Disposable {
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
			p.handlers.DoFinally(ctx, p.sig)
		}()
		p.handlers.DoSubscribe(ctx)
		var stop bool
		for {
			if stop {
				break
			}
			if p.rl != nil {
				p.rl.acquire(ctx, p.handlers)
			}
			select {
			case it, ok := <-p.source:
				if !ok {
					stop = true
					break
				}
				p.handlers.DoNextOrSuccess(ctx, it)
				consumer(ctx, it)
				p.handlers.DoAfterConsumer(ctx, it)
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
		default:
			panic("unreachable")
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

func (p *implFlux) release() {
	close(p.source)
	if p.rl != nil {
		_ = p.rl.Close()
	}
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

type rateLimiter struct {
	initN   int32
	tickets int32
	waiting chan struct{}
}

func (p *rateLimiter) Close() error {
	close(p.waiting)
	return nil
}

func (p *rateLimiter) reset(n uint32) {
	var v int32
	if n < math.MaxInt32 {
		v = int32(n)
	} else {
		v = math.MaxInt32
	}
	if atomic.CompareAndSwapInt32(&(p.tickets), 0, v) {
		p.waiting <- struct{}{}
	} else {
		atomic.AddInt32(&(p.tickets), v)
	}
}

func (p *rateLimiter) acquire(ctx context.Context, rxFunc RxFunc) {
	// tickets exhausted
	if atomic.LoadInt32(&(p.tickets)) == 0 {
		rxFunc.DoDrain(ctx)
		<-p.waiting
	}
	atomic.AddInt32(&(p.tickets), -1)
}

func newRateLimiter(n uint32) *rateLimiter {
	var v int32 = math.MaxInt32
	if n < math.MaxInt32 {
		v = int32(n)
	}
	return &rateLimiter{
		initN:   v,
		tickets: v,
		waiting: make(chan struct{}, 1),
	}
}
