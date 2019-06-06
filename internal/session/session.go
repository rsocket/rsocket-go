package session

import (
	"fmt"
	"time"

	"github.com/rsocket/rsocket-go/internal/socket"
)

type Session struct {
	index    int
	deadline time.Time
	socket   socket.ServerSocket
}

func (p *Session) Socket() socket.ServerSocket {
	return p.socket
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

func NewSession(deadline time.Time, sk socket.ServerSocket) *Session {
	return &Session{
		deadline: deadline,
		socket:   sk,
	}
}
