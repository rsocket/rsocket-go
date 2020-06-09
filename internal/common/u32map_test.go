package common_test

import (
	"sort"
	"sync/atomic"
	"testing"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/stretchr/testify/assert"
)

func TestU32map(t *testing.T) {
	var keys []int
	value := common.RandAlphanumeric(10)
	m := common.NewU32Map()
	for i := uint32(0); i < 10; i++ {
		m.Store(i, value)
		keys = append(keys, int(i))
	}
	v, ok := m.Load(1)
	assert.True(t, ok, "key not found")
	assert.Equal(t, value, v, "value doesn't match")

	_, ok = m.Load(10)
	assert.False(t, ok, "key should not exist")

	var keys2 []int
	m.Range(func(k uint32, _ interface{}) bool {
		keys2 = append(keys2, int(k))
		return true
	})
	sort.Ints(keys)
	sort.Ints(keys2)
	assert.Equal(t, keys, keys2, "keys doesn't match")

	m.Delete(1)
	_, ok = m.Load(1)
	assert.False(t, ok, "key should be deleted")

	var c int
	m.Range(func(k uint32, v interface{}) bool {
		c++
		return false
	})
	assert.Equal(t, 1, c, "should be 1")

	m.Clear()
	_, ok = m.Load(2)
	assert.False(t, ok, "should be closed already")
}

func BenchmarkU32Map(b *testing.B) {
	const value = "foobar"
	m := common.NewU32Map()
	next := uint32(0)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			m.Store(atomic.AddUint32(&next, 1)-1, value)
		}
	})
}
