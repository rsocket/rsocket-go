package rsocket

import (
	"container/list"
	"sync"
)

type Bus interface {
	Put(id string, first RSocket, others ...RSocket)
	Get(id string) (socket RSocket, ok bool)
	Remove(id string, socket RSocket) bool
}

func NewBus() Bus {
	return &implBus{
		mutex: &sync.RWMutex{},
		m:     make(map[string]*list.List, 0),
	}
}

type implBus struct {
	mutex *sync.RWMutex
	m     map[string]*list.List
}

func (p *implBus) Put(id string, first RSocket, others ...RSocket) {
	p.mutex.Lock()
	root, ok := p.m[id]
	if !ok {
		root = list.New()
		p.m[id] = root
	}
	root.PushBack(first)
	for _, it := range others {
		root.PushBack(it)
	}
	p.mutex.Unlock()
}

func (p *implBus) Get(id string) (socket RSocket, ok bool) {
	p.mutex.RLock()
	var root *list.List
	root, ok = p.m[id]
	if !ok {
		return
	}
	ok = root.Len() > 0
	if !ok {
		return
	}
	first := root.Front()
	socket = first.Value.(RSocket)
	root.MoveToBack(first)
	p.mutex.RUnlock()
	return
}

func (p *implBus) Remove(id string, socket RSocket) (ok bool) {
	p.mutex.Lock()
	var li *list.List
	li, ok = p.m[id]
	if !ok {
		p.mutex.Unlock()
		return
	}
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
	p.mutex.Unlock()
	return
}
