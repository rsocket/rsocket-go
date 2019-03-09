package rsocket

import (
	"context"
	"math"
	"sync/atomic"
)

type bQueue struct {
	rate       int32
	tickets    int32
	data       chan Payload
	breaker    chan struct{}
	onRequestN func(int32)
	polling    bool
}

func (p *bQueue) SetRate(n int32) {
	atomic.StoreInt32(&(p.rate), n)
	atomic.StoreInt32(&(p.tickets), 0)
}

func (p *bQueue) Close() error {
	//p.tickets = math.MaxInt32
	//close(p.breaker)
	close(p.data)
	return nil
}

func (p *bQueue) RequestN(n int32) {
	if n < 0 {
		return
	}
	if p.onRequestN != nil {
		defer p.onRequestN(n)
	}
	if atomic.CompareAndSwapInt32(&(p.tickets), 0, n) {
		if p.polling {
			p.breaker <- struct{}{}
		}
	} else {
		atomic.StoreInt32(&(p.tickets), n)
	}
}

func (p *bQueue) Add(payload Payload) (err error) {
	defer func() {
		if e, ok := recover().(error); ok {
			err = e
		}
	}()
	p.data <- payload
	return
}

func (p *bQueue) Poll(ctx context.Context) (payload Payload, ok bool) {
	p.polling = true
	defer func() {
		p.polling = false
	}()
	select {
	case <-ctx.Done():
		return nil, false
	default:
		// tickets exhausted
		foo := atomic.LoadInt32(&(p.tickets))
		if foo == math.MaxInt32 {
			v, ok := <-p.data
			return v, ok
		}
		if foo == 0 {
			if n := atomic.LoadInt32(&(p.rate)); n > 0 {
				p.RequestN(n)
			}
			<-p.breaker
		}
		if atomic.LoadInt32(&(p.tickets)) != math.MaxInt32 {
			atomic.AddInt32(&(p.tickets), -1)
		}
		v, ok := <-p.data
		return v, ok
	}
}

func newQueue(cap int, tickets int32, rate int32) *bQueue {
	return &bQueue{
		rate:    rate,
		tickets: tickets,
		data:    make(chan Payload, cap),
		breaker: make(chan struct{}, 1),
	}
}
