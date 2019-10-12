package transport

import (
	"go.uber.org/atomic"
)

// Counter represents a counter of read/write bytes.
type Counter struct {
	r, w *atomic.Uint64
}

// ReadBytes returns the number of bytes that have been read.
func (p Counter) ReadBytes() uint64 {
	return p.r.Load()
}

// WriteBytes returns the number of bytes that have been written.
func (p Counter) WriteBytes() uint64 {
	return p.w.Load()
}

func (p Counter) incrWriteBytes(n int) {
	p.w.Add(uint64(n))
}

func (p Counter) incrReadBytes(n int) {
	p.r.Add(uint64(n))
}

// NewCounter returns a new counter.
func NewCounter() *Counter {
	return &Counter{
		r: atomic.NewUint64(0),
		w: atomic.NewUint64(0),
	}
}
