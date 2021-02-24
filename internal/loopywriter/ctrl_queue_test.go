package loopywriter_test

import (
	"io"
	"testing"
	"time"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/loopywriter"
	"github.com/stretchr/testify/assert"
)

type fakeFrame struct {
}

func (f fakeFrame) Header() core.FrameHeader {
	panic("implement me")
}

func (f fakeFrame) Len() int {
	panic("implement me")
}

func (f fakeFrame) WriteTo(w io.Writer) (n int64, err error) {
	panic("implement me")
}

func (f fakeFrame) Done() {
	panic("implement me")
}

func (f fakeFrame) HandleDone(f2 func()) {
	panic("implement me")
}

func TestCtrlQueue(t *testing.T) {
	done := make(chan struct{})
	q := loopywriter.NewCtrlQueue(done)
	for i := 0; i < 100; i++ {
		q.Enqueue(fakeFrame{})
	}
	var cnt int
	q.Dispose(func(frame core.WriteableFrame) {
		cnt++
	})
	assert.Equal(t, 100, cnt)
}

func TestCtrlQueue_Dequeue(t *testing.T) {
	done := make(chan struct{})
	q := loopywriter.NewCtrlQueue(done)
	begin := time.Now()
	timeout := 500 * time.Millisecond
	_, _, err := q.Dequeue(true, timeout)
	assert.Equal(t, loopywriter.ErrDequeueTimeout, err, "should be timeout error")
	assert.True(t, time.Since(begin) >= timeout, "should be at least %s", timeout)
}

func BenchmarkCtrlQueue_Enqueue(b *testing.B) {
	done := make(chan struct{})
	q := loopywriter.NewCtrlQueue(done)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			q.Enqueue(fakeFrame{})
			_, _, _ = q.Dequeue(true, 3*time.Second)
		}
	})
	b.StartTimer()
}
