package session_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/rsocket/rsocket-go/internal/session"
	"github.com/rsocket/rsocket-go/internal/socket"
	"github.com/stretchr/testify/assert"
)

func TestSession(t *testing.T) {
	const total = 100
	var tokens []string
	manager := session.NewManager()
	for i := 0; i < total; i++ {
		deadline := time.Now().Add(time.Duration(i+1) * time.Second)
		token := fmt.Sprintf("token_%d", i)
		tokens = append(tokens, token)
		manager.Push(session.NewSession(deadline, socket.NewResumableServerSocket(nil, []byte(token))))
	}

	for _, token := range tokens {
		s, ok := manager.Load([]byte(token))
		assert.True(t, ok)
		assert.NotNil(t, s, "session is nil")
	}

	firstToken := []byte(tokens[0])
	s, ok := manager.Remove(firstToken)
	assert.True(t, ok)
	assert.NotNil(t, s)
	assert.Equal(t, firstToken, s.Token())
	assert.Equal(t, len(tokens)-1, manager.Len())
	for i := 1; i < len(tokens); i++ {
		manager.Pop()
	}
	assert.Equal(t, 0, manager.Len())
}
