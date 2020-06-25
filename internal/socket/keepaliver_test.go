package socket_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/rsocket/rsocket-go/internal/socket"
	"github.com/stretchr/testify/assert"
)

func TestKeepaliver(t *testing.T) {
	k := socket.NewKeepaliver(100 * time.Millisecond)

	time.AfterFunc(time.Second+50*time.Millisecond, func() {
		k.Stop()
		// stop again
		k.Stop()
	})

	beats := 0
L:
	for {
		select {
		case v := <-k.C():
			fmt.Println(v)
			beats++
		case <-k.Done():
			break L
		}
	}
	assert.Equal(t, 10, beats, "beats should be 10")
}
