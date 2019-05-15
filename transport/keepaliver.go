package transport

import "time"

type keepaliver struct {
	ticker    *time.Ticker
	interval  time.Duration
	lifetime  time.Duration
	heartbeat time.Time
}

func (p *keepaliver) Heartbeat() {
	p.heartbeat = time.Now()
}

func (p *keepaliver) IsDead(t time.Time) bool {
	return t.Sub(p.heartbeat) > p.lifetime
}

func (p *keepaliver) C() <-chan time.Time {
	return p.ticker.C
}

func (p *keepaliver) Close() (err error) {
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

func newKeepaliver(interval time.Duration, lifetime time.Duration) *keepaliver {
	return &keepaliver{
		ticker:    time.NewTicker(interval),
		heartbeat: time.Now(),
		lifetime:  lifetime,
	}
}
