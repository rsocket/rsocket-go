package rx

import (
	"context"
	"io"
	"math"
	"sync"
	"sync/atomic"

	. "github.com/rsocket/rsocket-go/payload"
)

const requestInfinite = math.MaxInt32

type rQueue interface {
	io.Closer
	Push(elem Payload) error
	Request(n int32)
	Poll(ctx context.Context) (elem Payload, ok bool)
	HandleRequest(handler func(n int32))
	SetRate(rate int32)
	SetTickets(n int32)
	Tickets() int32
}

type rQueueImpl struct {
	elements   chan Payload
	cond       *sync.Cond
	rate       int32
	tickets    int32
	onRequestN func(int32)
}

func (p *rQueueImpl) Close() (err error) {
	close(p.elements)
	return
}

func (p *rQueueImpl) HandleRequest(handler func(n int32)) {
	p.onRequestN = handler
}

func (p *rQueueImpl) SetRate(rate int32) {
	atomic.StoreInt32(&(p.rate), rate)
}

func (p *rQueueImpl) SetTickets(n int32) {
	atomic.StoreInt32(&(p.tickets), n)
}

func (p *rQueueImpl) Tickets() (n int32) {
	n = atomic.LoadInt32(&(p.tickets))
	if n < 0 {
		n = 0
	}
	return
}

func (p *rQueueImpl) Push(in Payload) (err error) {
	p.elements <- in
	return
}

func (p *rQueueImpl) Request(n int32) {
	p.doRequest(n, true)
}

func (p *rQueueImpl) doRequest(n int32, checkCond bool) {
	if n < 1 {
		return
	}
	if checkCond {
		p.cond.L.Lock()
	}
	if atomic.LoadInt32(&(p.tickets)) < 1 {
		atomic.StoreInt32(&(p.tickets), n)
		if checkCond {
			p.cond.Signal()
		}
	} else {
		if v := atomic.AddInt32(&(p.tickets), n); v == requestInfinite || v < 1 {
			atomic.StoreInt32(&(p.tickets), requestInfinite-1)
		}
	}
	if p.onRequestN != nil {
		p.onRequestN(n)
	}
	if checkCond {
		p.cond.L.Unlock()
	}
}

func (p *rQueueImpl) Poll(ctx context.Context) (pa Payload, ok bool) {
	select {
	case <-ctx.Done():
		return
	default:
		left := atomic.LoadInt32(&(p.tickets))
		if left == math.MaxInt32 {
			pa, ok = <-p.elements
			return
		}
		p.cond.L.Lock()
		for atomic.AddInt32(&(p.tickets), -1) < 0 {
			if r := p.rate; r > 0 {
				p.doRequest(r, false)
				continue
			}
			p.cond.Wait()
		}
		pa, ok = <-p.elements
		p.cond.L.Unlock()
	}
	return
}

func newQueue(cap int, tickets int32, rate int32) rQueue {
	return &rQueueImpl{
		cond:     sync.NewCond(&sync.Mutex{}),
		tickets:  tickets,
		elements: make(chan Payload, cap),
		rate:     rate,
	}
}
