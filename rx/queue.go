package rx

import (
	"context"
	"math"
	"sync/atomic"

	"github.com/rsocket/rsocket-go/payload"
)

const requestInfinite = math.MaxInt32

type bQueue struct {
	rate       int32
	tickets    int32
	data       chan payload.Payload
	breaker    chan struct{}
	onRequestN func(int32)
	polling    bool
}

func (p *bQueue) setRate(init, rate int32) {
	atomic.StoreInt32(&(p.rate), rate)
	atomic.StoreInt32(&(p.tickets), init)
}

func (p *bQueue) Close() error {
	close(p.data)
	return nil
}

func (p *bQueue) requestN(n int32) {
	if n < 0 {
		return
	}
	if !atomic.CompareAndSwapInt32(&(p.tickets), 0, n) {
		atomic.StoreInt32(&(p.tickets), n)
	} else if p.polling {
		p.breaker <- struct{}{}
	}
	if p.onRequestN != nil {
		p.onRequestN(n)
	}
}

func (p *bQueue) add(elem payload.Payload) (err error) {
	defer func() {
		if e, ok := recover().(error); ok {
			err = e
		}
	}()
	p.data <- elem
	return
}

func (p *bQueue) poll(ctx context.Context) (elem payload.Payload, ok bool) {
	p.polling = true
	defer func() {
		p.polling = false
	}()
	select {
	case <-ctx.Done():
		return
	default:
		tickets := atomic.LoadInt32(&(p.tickets))
		if tickets == requestInfinite {
			elem, ok = <-p.data
			return
		}
		// tickets exhausted
		if tickets == 0 {
			if n := atomic.LoadInt32(&(p.rate)); n > 0 {
				p.requestN(n)
			}
			<-p.breaker
		}
		// decrease a ticket
		if atomic.LoadInt32(&(p.tickets)) != requestInfinite {
			atomic.AddInt32(&(p.tickets), -1)
		}
		elem, ok = <-p.data
		return
	}
}

func newQueue(cap int, tickets int32, rate int32) *bQueue {
	return &bQueue{
		rate:    rate,
		tickets: tickets,
		data:    make(chan payload.Payload, cap),
		breaker: make(chan struct{}, 1),
	}
}
