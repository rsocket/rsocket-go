package buffer_test

import (
	"strings"
	"sync/atomic"
	"testing"

	"github.com/rsocket/rsocket-go/internal/buffer"
	"github.com/stretchr/testify/assert"
)

func TestNewUnbounded(t *testing.T) {
	u := buffer.NewUnbounded()

	go func() {
		assert.True(t, u.Put("foo"))
		assert.True(t, u.Put("bar"))
		assert.True(t, u.Put("qux"))

		u.Dispose()
		assert.False(t, u.Put("must failed"))
	}()

	done := make(chan struct{})
	var read []string

	go func() {
		defer close(done)
		for next := range u.Get() {
			read = append(read, next.(string))
			u.Load()
		}
	}()

	<-done

	assert.Equal(t, "foo,bar,qux", strings.Join(read, ","), "result doesn't match")
}

func TestEmptyUnbounded(t *testing.T) {
	u := buffer.NewUnbounded()

	done := make(chan struct{})
	cnt := new(int32)

	go func() {
		defer close(done)
		for range u.Get() {
			atomic.AddInt32(cnt, 1)
			u.Load()
		}
	}()

	go func() {
		u.Dispose()
	}()

	<-done

	assert.Zero(t, atomic.LoadInt32(cnt))
}
