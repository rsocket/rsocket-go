package rsocket

import (
	"container/heap"
	"fmt"
	"sync"
	"time"

	"github.com/rsocket/rsocket-go/socket"
)

type Session struct {
	index    int
	deadline time.Time
	socket   socket.ServerSocket
}

func (p *Session) Close() error {
	return p.socket.Close()
}

func (p *Session) IsDead() (dead bool) {
	dead = time.Now().After(p.deadline)
	return
}

func (p *Session) String() string {
	tk, _ := p.socket.Token()
	return fmt.Sprintf("Session{token=0x%02X,deadline=%d}", tk, p.deadline.Unix())
}

func (p *Session) Token() (token []byte) {
	token, _ = p.socket.Token()
	return
}

type SessionHeap []*Session

func (h SessionHeap) Len() int {
	return len(h)
}

func (h SessionHeap) Less(i, j int) bool {
	return h[i].deadline.Before(h[j].deadline)
}

func (h SessionHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index, h[j].index = i, j
}

func (h *SessionHeap) Push(x interface{}) {
	session := x.(*Session)
	session.index = len(*h)
	*h = append(*h, session)
}

func (h *SessionHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	old[n-1] = nil
	*h = old[0 : n-1]
	x.index = -1
	return x
}

type SessionManager struct {
	locker *sync.RWMutex
	h      *SessionHeap
	m      map[string]*Session
}

func (p *SessionManager) Len() (n int) {
	p.locker.RLock()
	n = len(*p.h)
	p.locker.RUnlock()
	return
}

func (p *SessionManager) Push(session *Session) {
	p.locker.Lock()
	heap.Push(p.h, session)
	p.m[(string)(session.Token())] = session
	p.locker.Unlock()
}

func (p *SessionManager) Load(token []byte) (session *Session, ok bool) {
	p.locker.RLock()
	session, ok = p.m[(string)(token)]
	p.locker.RUnlock()
	return
}

func (p *SessionManager) Remove(token []byte) (session *Session, ok bool) {
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

func (p *SessionManager) Pop() (session *Session) {
	p.locker.Lock()
	session = heap.Pop(p.h).(*Session)
	if session != nil {
		delete(p.m, (string)(session.Token()))
	}
	p.locker.Unlock()
	return
}

func NewSession(deadline time.Time, sk socket.ServerSocket) *Session {
	return &Session{
		deadline: deadline,
		socket:   sk,
	}
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		locker: &sync.RWMutex{},
		h:      &SessionHeap{},
		m:      make(map[string]*Session),
	}
}
