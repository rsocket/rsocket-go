package socket

import (
	"crypto/rand"
	"encoding/binary"
	"sync"
	"testing"
)

func nextID() uint32 {
	b4 := make([]byte, 4)
	_, _ = rand.Read(b4)
	return uint32(maskStreamID) & binary.BigEndian.Uint32(b4)
}

func BenchmarkLock(b *testing.B) {
	var v reqRC
	m := make(map[uint32]interface{})
	var lk sync.RWMutex
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := nextID()
			switch id % 3 {
			case 1:
				lk.Lock()
				m[id] = v
				lk.Unlock()
			case 2:
				lk.RLock()
				_ = m[id]
				lk.RUnlock()
			default:
				lk.Lock()
				delete(m, id)
				lk.Unlock()
			}
		}
	})
}

func BenchmarkSync(b *testing.B) {
	var v reqRC
	m := &sync.Map{}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			id := nextID()
			switch id % 3 {
			case 1:
				m.Store(id, v)
			case 2:
				m.Load(id)
			default:
				m.Delete(id)
			}
		}
	})

}
