package rsocket

import (
	"log"
	"testing"
	"time"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/socket"
)

func TestSession(t *testing.T) {
	manager := NewSessionManager()
	for i := 0; i < 3; i++ {
		deadline := time.Now().Add(time.Duration(common.RandIntn(30)) * time.Second)
		token := common.RandAlphanumeric(32)
		manager.Push(NewSession(deadline, socket.NewServerResume(nil, []byte(token))))
	}

	for _, value := range *(manager.h) {
		session, ok := manager.Load(value.Token())
		log.Printf("session=%s,ok=%t\n", session, ok)
	}

	for manager.Len() > 0 {
		log.Println("session:", manager.Pop())
	}
}
