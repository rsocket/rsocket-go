package session

import (
	"container/heap"
	"sync"
)

type Manager struct {
	locker *sync.RWMutex
	h      *sHeap
	m      map[string]*Session
}

func (p *Manager) Len() (n int) {
	p.locker.RLock()
	n = len(*p.h)
	p.locker.RUnlock()
	return
}

func (p *Manager) Push(session *Session) {
	p.locker.Lock()
	heap.Push(p.h, session)
	p.m[(string)(session.Token())] = session
	p.locker.Unlock()
}

func (p *Manager) Load(token []byte) (session *Session, ok bool) {
	p.locker.RLock()
	session, ok = p.m[(string)(token)]
	p.locker.RUnlock()
	return
}

func (p *Manager) Remove(token []byte) (session *Session, ok bool) {
	p.locker.Lock()
	session, ok = p.m[(string)(token)]
	if ok && session.index > -1 {
		heap.Remove(p.h, session.index)
		delete(p.m, (string)(token))
		session.index = -1
	}
	p.locker.Unlock()
	return
}

func (p *Manager) Pop() (session *Session) {
	p.locker.Lock()
	session = heap.Pop(p.h).(*Session)
	if session != nil {
		delete(p.m, (string)(session.Token()))
	}
	p.locker.Unlock()
	return
}

func NewManager() *Manager {
	return &Manager{
		locker: &sync.RWMutex{},
		h:      &sHeap{},
		m:      make(map[string]*Session),
	}
}
