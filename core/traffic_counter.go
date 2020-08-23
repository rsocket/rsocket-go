package core

import (
	"go.uber.org/atomic"
)

// TrafficCounter represents a counter of read/write bytes.
type TrafficCounter struct {
	r, w *atomic.Uint64
}

// ReadBytes returns the number of bytes that have been read.
func (p TrafficCounter) ReadBytes() uint64 {
	return p.r.Load()
}

// WriteBytes returns the number of bytes that have been written.
func (p TrafficCounter) WriteBytes() uint64 {
	return p.w.Load()
}

// IncWriteBytes increases bytes wrote.
func (p TrafficCounter) IncWriteBytes(n int) {
	p.w.Add(uint64(n))
}

// IncReadBytes increases bytes read.
func (p TrafficCounter) IncReadBytes(n int) {
	p.r.Add(uint64(n))
}

// NewTrafficCounter returns a new counter.
func NewTrafficCounter() *TrafficCounter {
	return &TrafficCounter{
		r: atomic.NewUint64(0),
		w: atomic.NewUint64(0),
	}
}
