package stopwatch

import (
	"sync"
	"time"
)

type Stopwatch interface {
	Reset()
	Checkpoint() time.Duration
}

func NewStopwatch() Stopwatch {
	return &stopwatch{
		prev: time.Now(),
	}
}

var _ Stopwatch = (*stopwatch)(nil)

type stopwatch struct {
	mu   sync.Mutex
	prev time.Time
}

func (sw *stopwatch) Reset() {
	sw.mu.Lock()
	sw.prev = time.Now()
	sw.mu.Unlock()
}

func (sw *stopwatch) Checkpoint() time.Duration {
	now := time.Now()
	sw.mu.Lock()
	prev := sw.prev
	sw.prev = now
	sw.mu.Unlock()
	return now.Sub(prev)
}
