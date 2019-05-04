package socket

import (
	"time"
)

type keepaliver struct {
	ticker   *time.Ticker
	interval time.Duration
}

func (p *keepaliver) C() <-chan time.Time {
	return p.ticker.C
}

func (p *keepaliver) Stop() {
	if p.ticker != nil {
		p.ticker.Stop()
	}
	return
}

func (p *keepaliver) Reset(interval time.Duration) {
	if interval != p.interval {
		p.ticker.Stop()
		p.ticker = time.NewTicker(interval)
	}
	return
}

func newKeepaliver(interval time.Duration) *keepaliver {
	return &keepaliver{
		interval: interval,
		ticker:   time.NewTicker(interval),
	}
}
