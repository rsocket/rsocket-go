package socket

import (
	"time"
)

type Keepaliver struct {
	ticker *time.Ticker
	done   chan struct{}
}

func (p Keepaliver) C() <-chan time.Time {
	return p.ticker.C
}

func (p Keepaliver) Done() <-chan struct{} {
	return p.done
}

func (p Keepaliver) Stop() {
	defer func() {
		_ = recover()
	}()
	p.ticker.Stop()
	close(p.done)
}

func NewKeepaliver(interval time.Duration) *Keepaliver {
	return &Keepaliver{
		ticker: time.NewTicker(interval),
		done:   make(chan struct{}),
	}
}
