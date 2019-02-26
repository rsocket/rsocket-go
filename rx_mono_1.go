package rsocket

import (
	"context"
	"sync"
)

type implMono struct {
	init     MonoCreate
	ss       Scheduler
	result   interface{}
	locker   *sync.Mutex
	sig      SignalType
	done     chan struct{}
	handlers RxFunc
}

func (p *implMono) onAfterSubscribe(fn Consumer) Mono {
	p.locker.Lock()
	p.handlers.RegisterAfterConsumer(fn)
	p.locker.Unlock()
	return p
}

func (p *implMono) DoOnCancel(onCancel OnCancel) Mono {
	p.locker.Lock()
	p.handlers.RegisterCancel(onCancel)
	p.locker.Unlock()
	return p
}

func (p *implMono) DoOnError(onError OnError) Mono {
	p.locker.Lock()
	p.handlers.RegisterError(onError)
	p.locker.Unlock()
	return p
}

func (p *implMono) DoOnNext(onNext Consumer) Mono {
	p.locker.Lock()
	p.handlers.RegisterNext(onNext)
	p.locker.Unlock()
	return p
}

func (p *implMono) DoOnSuccess(onSuccess Consumer) Mono {
	p.locker.Lock()
	p.handlers.RegisterSuccess(onSuccess)
	p.locker.Unlock()
	return p
}

func (p *implMono) DoFinally(fn OnFinally) Mono {
	p.locker.Lock()
	p.handlers.RegisterFinally(fn)
	p.locker.Unlock()
	return p
}

func (p *implMono) SubscribeOn(scheduler Scheduler) Publisher {
	p.ss = scheduler
	return p
}

func (p *implMono) Subscribe(ctx context.Context, consumer Consumer) Disposable {
	cc, cancel := context.WithCancel(ctx)
	p.ss.Do(cc, func(ctx context.Context) {
		defer func() {
			if err, ok := recover().(error); ok {
				p.Error(err)
			}
		}()
		if p.init != nil {
			p.init(cc, p)
		}
		// waiting for subscribe
		<-p.done
		defer p.handlers.DoFinally(ctx, p.sig)
		switch p.sig {
		case SigError:
			p.handlers.DoError(ctx, p.result.(error))
		case SigSuccess:
			payload := p.result.(Payload)
			p.handlers.DoNextOrSuccess(ctx, payload)
			consumer(ctx, payload)
			p.handlers.DoAfterConsumer(ctx, payload)
		case SigCancel:
			p.handlers.DoCancel(ctx)
			cancel()
		default:
			panic("unreachable")
		}
	})
	return p
}

func (p *implMono) Success(payload Payload) {
	p.locker.Lock()
	defer p.locker.Unlock()
	if p.sig != SigReady {
		return
	}
	p.result = payload
	p.sig = SigSuccess
	close(p.done)
}

func (p *implMono) Error(e error) {
	p.locker.Lock()
	defer p.locker.Unlock()
	if p.sig != SigReady {
		return
	}
	p.result = e
	p.sig = SigError
	close(p.done)
}

func (p *implMono) Dispose() {
	p.locker.Lock()
	defer p.locker.Unlock()
	if p.sig != SigReady {
		return
	}
	p.sig = SigCancel
	close(p.done)
}
