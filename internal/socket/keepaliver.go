package socket

import (
	"time"
)

// Keepaliver controls connection keepalive.
type Keepaliver struct {
	ticker *time.Ticker
	done   chan struct{}
}

// C returns ticker.C.
func (p Keepaliver) C() <-chan time.Time {
	return p.ticker.C
}

// Done returns done chan.
func (p Keepaliver) Done() <-chan struct{} {
	return p.done
}

// Stop stops keepaliver.
func (p Keepaliver) Stop() {
	defer func() {
		_ = recover()
	}()
	p.ticker.Stop()
	close(p.done)
}

// NewKeepaliver creates a new keepaliver.
func NewKeepaliver(interval time.Duration) *Keepaliver {
	return &Keepaliver{
		ticker: time.NewTicker(interval),
		done:   make(chan struct{}),
	}
}
