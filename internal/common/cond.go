package common

import (
	"context"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Cond
// see https://gist.github.com/zviadm/c234426882bfc8acba88f3503edaaa36#file-cond2-go
type Cond struct {
	L sync.Locker
	n unsafe.Pointer
}

func NewCond(l sync.Locker) *Cond {
	c := &Cond{
		L: l,
	}
	n := make(chan struct{})
	c.n = unsafe.Pointer(&n)
	return c
}

func (c *Cond) NotifyChan() <-chan struct{} {
	ptr := atomic.LoadPointer(&c.n)
	return *((*chan struct{})(ptr))
}

func (c *Cond) Broadcast() {
	n := make(chan struct{})
	ptrOld := atomic.SwapPointer(&c.n, unsafe.Pointer(&n))
	close(*(*chan struct{})(ptrOld))
}

func (c *Cond) Wait(ctx context.Context) (isCtx bool) {
	n := c.NotifyChan()
	c.L.Unlock()
	select {
	case <-n:
	case <-ctx.Done():
		isCtx = true
	}
	c.L.Lock()
	return
}
