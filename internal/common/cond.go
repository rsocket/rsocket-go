package common

import (
	"context"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Cond is a customized Cond which support context.
// See https://gist.github.com/zviadm/c234426882bfc8acba88f3503edaaa36#file-cond2-go
type Cond struct {
	L sync.Locker
	n unsafe.Pointer
}

// NotifyChan returns readonly notify chan.
func (c *Cond) NotifyChan() <-chan struct{} {
	ptr := atomic.LoadPointer(&c.n)
	return *((*chan struct{})(ptr))
}

// Broadcast broadcast Cond.
func (c *Cond) Broadcast() {
	n := make(chan struct{})
	ptrOld := atomic.SwapPointer(&c.n, unsafe.Pointer(&n))
	close(*(*chan struct{})(ptrOld))
}

// Wait waits Cond.
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

// NewCond creates a new Cond.
func NewCond(l sync.Locker) *Cond {
	c := &Cond{
		L: l,
	}
	n := make(chan struct{})
	c.n = unsafe.Pointer(&n)
	return c
}
