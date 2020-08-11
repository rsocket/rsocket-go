package socket

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientStreamIDs_Next(t *testing.T) {
	ids := clientStreamIDs{}
	id, firstLap := ids.Next()
	assert.Equal(t, uint32(1), id)
	assert.True(t, firstLap)
}

func TestServerStreamIDs_Next(t *testing.T) {
	ids := serverStreamIDs{}
	id, firstLap := ids.Next()
	assert.Equal(t, uint32(2), id)
	assert.True(t, firstLap)
}

func BenchmarkServerStreamIDs_Next(b *testing.B) {
	ids := serverStreamIDs{}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ids.Next()
		}
	})
}

func BenchmarkClientStreamIDs_Next(b *testing.B) {
	ids := clientStreamIDs{}
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ids.Next()
		}
	})
}
