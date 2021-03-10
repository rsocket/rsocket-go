package map32

import (
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func TestMap32_Destroy(t *testing.T) {
	m := New()
	const cnt = 10000
	for i := 0; i < cnt; i++ {
		m.Store(uint32(i), time.Now())
	}

	m.Destroy()

	_, ok := m.Load(1)
	assert.False(t, ok)
}

func TestNil(t *testing.T) {
	var s *shard32
	s.Load(1)
	s.Delete(1)
	s.Range(func(u uint32, i interface{}) bool {
		return true
	})
	s.Store(1, 1)
}

func TestEmpty(t *testing.T) {
	m := New(WithCap(8), WithHasher(func(key uint32, cap int) (offset int) {
		offset = int(key) % cap
		return
	}))

	m.Load(1)
	m.Range(func(u uint32, i interface{}) bool {
		return true
	})
	m.Delete(1)
	m.Store(1, 1)
	m.Load(1)
}

func TestAll(t *testing.T) {
	m := New(WithCap(16))
	const cnt = 100000
	var wg sync.WaitGroup
	wg.Add(cnt)
	for i := 0; i < cnt; i++ {
		go func(n uint32) {
			m.Store(n, struct{}{})
			wg.Done()
		}(uint32(i))
	}
	wg.Wait()

	wg.Add(cnt)
	for i := 0; i < cnt; i++ {
		go func(n uint32) {
			defer wg.Done()
			_, ok := m.Load(n)
			assert.True(t, ok)
		}(uint32(i))
	}
	wg.Wait()

	var n int
	m.Range(func(u uint32, i interface{}) bool {
		n++
		return true
	})
	assert.Equal(t, cnt, n)

	wg.Add(cnt)
	for i := 0; i < cnt; i++ {
		go func(n uint32) {
			defer wg.Done()
			m.Delete(n)
		}(uint32(i))
	}
	wg.Wait()
	_, ok := m.Load(1)
	assert.False(t, ok)

	_, ok = m.Load(cnt - 1)
	assert.False(t, ok)

}

func BenchmarkMap32(b *testing.B) {
	m := New()
	const cnt = 1000000
	empty := struct{}{}
	for i := 0; i < cnt; i++ {
		m.Store(uint32(i), empty)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m.Delete(uint32(rand.Intn(cnt)))
		}
	})
}
