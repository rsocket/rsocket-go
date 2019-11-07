package lease

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var (
	ErrLeaseNotRcv         = errors.New("lease was not received yet")
	ErrLeaseExpired        = errors.New("lease expired")
	ErrLeaseNoMoreRequests = errors.New("no more lease")
)

type Leases interface {
	Next(ctx context.Context) (ch chan Lease, ok bool)
}

type Lease struct {
	TimeToLive       time.Duration
	NumberOfRequests uint32
	Metadata         []byte
}

func NewSimpleLease(interval, ttl, delay time.Duration, numberOfRequest uint32) (Leases, error) {
	if interval <= 0 {
		return nil, fmt.Errorf("invalid simple lease interval: %s", interval)
	}
	if ttl <= 0 {
		return nil, fmt.Errorf("invalid simple lease TTL: %s", ttl)
	}
	if numberOfRequest < 1 {
		return nil, fmt.Errorf("invalid simple lease NumberOfRequest: %d", numberOfRequest)
	}
	return simpleLease{
		delay:    delay,
		ttl:      ttl,
		n:        numberOfRequest,
		interval: interval,
	}, nil
}

type simpleLease struct {
	delay    time.Duration
	ttl      time.Duration
	n        uint32
	interval time.Duration
}

func (s simpleLease) Next(ctx context.Context) (chan Lease, bool) {
	ch := make(chan Lease)
	go func(ctx context.Context, ch chan Lease) {
		defer close(ch)

		if s.delay > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(s.delay):
				ch <- s.create()
			}
		}

		tk := time.NewTicker(s.interval)
		for {
			select {
			case <-ctx.Done():
				tk.Stop()
				return
			case <-tk.C:
				ch <- s.create()
			}
		}
	}(ctx, ch)
	return ch, true
}

func (s simpleLease) create() Lease {
	return Lease{
		TimeToLive:       s.ttl,
		NumberOfRequests: s.n,
	}
}
