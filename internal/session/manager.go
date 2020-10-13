package session

import (
	"container/heap"
	"sync"
)

// Manager is used to manage RSocket session when resume is enabled.
type Manager struct {
	sync.RWMutex
	h *sHeap
	m map[string]*Session
}

// Len returns size of session in current manager.
func (p *Manager) Len() (n int) {
	p.RLock()
	defer p.RUnlock()
	n = len(*p.h)
	return
}

// Push push a new session.
func (p *Manager) Push(session *Session) {
	p.Lock()
	defer p.Unlock()
	heap.Push(p.h, session)
	p.m[(string)(session.Token())] = session
}

// Load returns session with custom token.
func (p *Manager) Load(token []byte) (session *Session, ok bool) {
	p.RLock()
	defer p.RUnlock()
	session, ok = p.m[(string)(token)]
	return
}

// Remove remove a session with custom token.
func (p *Manager) Remove(token []byte) (session *Session, ok bool) {
	p.Lock()
	defer p.Unlock()
	session, ok = p.m[(string)(token)]
	if ok && session.index > -1 {
		heap.Remove(p.h, session.index)
		delete(p.m, (string)(token))
		session.index = -1
	}
	return
}

// Pop pop earliest session.
func (p *Manager) Pop() (session *Session) {
	p.Lock()
	defer p.Unlock()
	session = heap.Pop(p.h).(*Session)
	if session != nil {
		delete(p.m, (string)(session.Token()))
	}
	return
}

// NewManager returns a new blank session manager.
func NewManager() *Manager {
	return &Manager{
		h: &sHeap{},
		m: make(map[string]*Session),
	}
}
