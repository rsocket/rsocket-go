package loopywriter

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/timerpool"
	"github.com/rsocket/rsocket-go/lease"
)

const _queueThreshold = 1000

var ErrDequeueTimeout = errors.New("dequeue timeout")

var _itemNodePool = sync.Pool{
	New: func() interface{} {
		return new(itemNode)
	},
}

type itemNode struct {
	it   core.WriteableFrame
	next *itemNode
}

type itemList struct {
	head *itemNode
	tail *itemNode
}

func borrowItemNode(i core.WriteableFrame) *itemNode {
	n := _itemNodePool.Get().(*itemNode)
	n.it = i
	return n
}

func returnItemNode(node *itemNode) {
	node.next = nil
	node.it = nil
	_itemNodePool.Put(node)
}

func (il *itemList) enqueue(i core.WriteableFrame) {
	n := borrowItemNode(i)
	if il.tail == nil {
		il.head, il.tail = n, n
		return
	}
	il.tail.next = n
	il.tail = n
}

func (il *itemList) dequeue() core.WriteableFrame {
	if il.head == nil {
		return nil
	}
	cur := il.head
	value := cur.it
	il.head = cur.next
	if il.head == nil {
		il.tail = nil
	}
	returnItemNode(cur)
	return value
}

func (il *itemList) dequeueAll() *itemNode {
	h := il.head
	il.head, il.tail = nil, nil
	return h
}

func (il *itemList) isEmpty() bool {
	return il.head == nil
}

type CtrlQueue struct {
	mu             sync.Mutex
	wake           chan struct{}
	items          *itemList
	consumeWaiting bool
	leaseChan      <-chan lease.Lease
	done           <-chan struct{}
	err            error
	full           bool
	fullCond       sync.Cond
	size           int32
}

func NewCtrlQueue(done <-chan struct{}) *CtrlQueue {
	q := &CtrlQueue{
		wake:  make(chan struct{}, 1),
		done:  done,
		items: &itemList{},
	}
	q.fullCond.L = &q.mu
	return q
}

func (c *CtrlQueue) BindLease(leaseChan <-chan lease.Lease) {
	c.mu.Lock()
	c.leaseChan = leaseChan
	c.mu.Unlock()
}

func (c *CtrlQueue) Enqueue(next core.WriteableFrame) (ok bool) {
	var wakeUp bool
	c.mu.Lock()

	if c.err != nil {
		c.mu.Unlock()
		return
	}

	if c.consumeWaiting {
		wakeUp = true
		c.consumeWaiting = false
	}

	select {
	case <-c.done:
	default:
		if c.full {
			c.fullCond.Wait()
		}
		c.items.enqueue(next)
		if n := atomic.AddInt32(&c.size, 1); n == _queueThreshold {
			c.full = true
		}
		ok = true
	}

	c.mu.Unlock()

	if wakeUp {
		c.wake <- struct{}{}
	}
	return
}

func (c *CtrlQueue) Dequeue(block bool, timeout time.Duration) (next core.WriteableFrame, le lease.Lease, err error) {
	var tc <-chan time.Time
	if timeout != 0 {
		timer := timerpool.Get(timeout)
		tc = timer.C
		defer timerpool.Put(timer)
	}
	for {
		select {
		case <-tc:
			err = ErrDequeueTimeout
			return
		case le = <-c.leaseChan:
			return
		default:
		}

		c.mu.Lock()
		if !c.items.isEmpty() {
			next = c.items.dequeue()
			if n := atomic.AddInt32(&c.size, -1); n == _queueThreshold-1 {
				c.full = false
				c.fullCond.Broadcast()
			}
			c.mu.Unlock()
			return
		}
		c.consumeWaiting = true
		c.mu.Unlock()

		if !block {
			return
		}

		select {
		case <-c.wake:
		case <-c.done:
			err = io.EOF
			return
		case <-tc:
			err = ErrDequeueTimeout
			return
		case le = <-c.leaseChan:
			return
		}
	}
}

func (c *CtrlQueue) Dispose(f DisposeFunc) {
	c.mu.Lock()
	if c.err != nil {
		c.mu.Unlock()
		return
	}
	c.err = io.EOF
	c.fullCond.Broadcast()

	head := c.items.dequeueAll()
	for head != nil {
		if f != nil {
			f(head.it)
		}

		next := head.next

		returnItemNode(head)

		head = next
	}
	c.mu.Unlock()
}
