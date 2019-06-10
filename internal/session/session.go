package session

import (
	"fmt"
	"time"

	"github.com/rsocket/rsocket-go/internal/socket"
)

// Session represents a lifecycle of a RSocket server socket.
type Session struct {
	index    int
	deadline time.Time
	socket   socket.ServerSocket
}

// Socket returns RSocket server socket in current session.
func (p *Session) Socket() socket.ServerSocket {
	return p.socket
}

// Close close current session.
func (p *Session) Close() error {
	return p.socket.Close()
}

// IsDead returns true if current session is dead.
func (p *Session) IsDead() (dead bool) {
	dead = time.Now().After(p.deadline)
	return
}

func (p *Session) String() string {
	tk, _ := p.socket.Token()
	return fmt.Sprintf("Session{token=0x%02X,deadline=%d}", tk, p.deadline.Unix())
}

// Token returns token of session.
func (p *Session) Token() (token []byte) {
	token, _ = p.socket.Token()
	return
}

// NewSession returns a new session.
func NewSession(deadline time.Time, sk socket.ServerSocket) *Session {
	return &Session{
		deadline: deadline,
		socket:   sk,
	}
}
