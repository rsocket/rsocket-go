package rsocket

import (
	"container/list"
	"context"
	"sync"
	"time"
)

const (
	busInitSleepTime = 4 * time.Millisecond
	busMaxSleepTime  = 100 * time.Millisecond
	busTryGetTimeout = 5 * time.Second
)

type Bus interface {
	Put(id string, first RSocket, others ...RSocket)
	Get(id string) (socket RSocket, ok bool)
	Remove(id string, socket RSocket) bool
}

func NewBus() Bus {
	return &implBus{
		m: &sync.Map{},
	}
}

type implBus struct {
	m *sync.Map
}

func (p *implBus) Put(id string, first RSocket, others ...RSocket) {
	v, _ := p.m.LoadOrStore(id, list.New())
	li := v.(*list.List)
	li.PushBack(first)
	for _, it := range others {
		li.PushBack(it)
	}
}

func (p *implBus) Get(id string) (socket RSocket, ok bool) {
	socket, ok = p.tryGet(id)
	if ok {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), busTryGetTimeout)
	defer func() {
		cancel()
	}()
	sleep := busInitSleepTime
	for {
		if ok {
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
			if sleep > busMaxSleepTime {
				time.Sleep(busMaxSleepTime)
			} else {
				time.Sleep(sleep)
				sleep += sleep >> 1
			}
			socket, ok = p.tryGet(id)
		}
	}
}

func (p *implBus) Remove(id string, socket RSocket) (ok bool) {
	var v interface{}
	v, ok = p.m.Load(id)
	if !ok {
		return
	}
	li := v.(*list.List)
	var cur *list.Element
	for cur = li.Front(); cur != nil; cur = cur.Next() {
		if cur.Value == socket {
			break
		}
	}
	ok = cur != nil
	if ok {
		li.Remove(cur)
	}
	return
}

func (p *implBus) tryGet(id string) (socket RSocket, ok bool) {
	var v interface{}
	v, ok = p.m.Load(id)
	if !ok {
		return
	}
	li := v.(*list.List)
	ok = li.Len() > 0
	if !ok {
		return
	}
	if li.Len() == 1 {
		socket = li.Front().Value.(RSocket)
		return
	}
	first := li.Front()
	socket = first.Value.(RSocket)
	li.MoveToBack(first)
	return
}
