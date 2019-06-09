package transport

import "sync/atomic"

// Counter represents a counter of read/write bytes.
type Counter struct {
	r, w uint64
}

// ReadBytes returns the number of bytes that have been read.
func (p *Counter) ReadBytes() uint64 {
	return atomic.LoadUint64(&(p.r))
}

// WriteBytes returns the number of bytes that have been written.
func (p *Counter) WriteBytes() uint64 {
	return atomic.LoadUint64(&(p.w))
}

func (p *Counter) incrWriteBytes(n int) {
	atomic.AddUint64(&(p.w), uint64(n))
}

func (p *Counter) incrReadBytes(n int) {
	atomic.AddUint64(&(p.r), uint64(n))
}

// NewCounter returns a new counter.
func NewCounter() *Counter {
	return &Counter{}
}
