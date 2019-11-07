package session

import (
	"container/heap"
	"sync"
)

// Manager is used to manage RSocket session when resume is enabled.
type Manager struct {
	locker sync.RWMutex
	h      *sHeap
	m      map[string]*Session
}

// Len returns size of session in current manager.
func (p *Manager) Len() (n int) {
	p.locker.RLock()
	n = len(*p.h)
	p.locker.RUnlock()
	return
}

// Push push a new session.
func (p *Manager) Push(session *Session) {
	p.locker.Lock()
	defer p.locker.Unlock()
	heap.Push(p.h, session)
	p.m[(string)(session.Token())] = session
}

// Load returns session with custom token.
func (p *Manager) Load(token []byte) (session *Session, ok bool) {
	p.locker.RLock()
	session, ok = p.m[(string)(token)]
	p.locker.RUnlock()
	return
}

// Remove remove a session with custom token.
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

// Pop pop earliest session.
func (p *Manager) Pop() (session *Session) {
	p.locker.Lock()
	session = heap.Pop(p.h).(*Session)
	if session != nil {
		delete(p.m, (string)(session.Token()))
	}
	p.locker.Unlock()
	return
}

// NewManager returns a new blank session manager.
func NewManager() *Manager {
	return &Manager{
		h: &sHeap{},
		m: make(map[string]*Session),
	}
}
