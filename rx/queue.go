package rx

import (
	"context"
	"errors"
	"math"
	"sync"
	"sync/atomic"

	. "github.com/rsocket/rsocket-go/payload"
)

const (
	RequestInfinite = math.MaxInt32

	defaultQueueSize = 16
)

var errIllegalCap = errors.New("cap must greater than zero")

type queue struct {
	elements   chan Payload
	cond       *sync.Cond
	tickets    int32
	onRequestN func(int32)
	done       chan struct{}
}

func (p *queue) Close() (err error) {
	p.cond.Broadcast()
	close(p.elements)
	return
}

func (p *queue) HandleRequest(handler func(n int32)) {
	p.onRequestN = handler
}

func (p *queue) SetTickets(n int32) {
	atomic.StoreInt32(&(p.tickets), n)
}

func (p *queue) Tickets() (n int32) {
	n = atomic.LoadInt32(&(p.tickets))
	if n < 0 {
		n = 0
	}
	return
}

func (p *queue) Push(in Payload) (err error) {
	defer func() {
		err, _ = recover().(error)
	}()
	p.elements <- in
	return
}

func (p *queue) Request(n int32) {
	if n < 1 {
		return
	}
	p.cond.L.Lock()
	if atomic.LoadInt32(&(p.tickets)) < 1 {
		atomic.StoreInt32(&(p.tickets), n)
		p.cond.Signal()
	} else {
		atomic.StoreInt32(&(p.tickets), n)
	}
	if p.onRequestN != nil {
		p.onRequestN(n)
	}
	p.cond.L.Unlock()
}

func (p *queue) Poll(ctx context.Context) (pa Payload, ok bool) {
	select {
	case <-ctx.Done():
		return
	case <-p.done:
		return
	default:
		p.cond.L.Lock()
		if atomic.LoadInt32(&(p.tickets)) == RequestInfinite {
			pa, ok = <-p.elements
			if !ok {
				close(p.done)
			}
			p.cond.L.Unlock()
			return
		}
		for atomic.AddInt32(&(p.tickets), -1) < 0 {
			p.cond.Wait()
		}
		pa, ok = <-p.elements
		if !ok {
			close(p.done)
		}
		p.cond.L.Unlock()
	}
	return
}

func newQueue(cap int, tickets int32) *queue {
	if cap < 1 {
		panic(errIllegalCap)
	}
	return &queue{
		cond:     sync.NewCond(&sync.Mutex{}),
		tickets:  tickets,
		elements: make(chan Payload, cap),
		done:     make(chan struct{}),
	}
}
