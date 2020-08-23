package lease

import (
	"context"
	"time"

	"github.com/pkg/errors"
)

// lease errors
var (
	ErrLeaseNotRcv         = errors.New("rsocket: lease was not received yet")
	ErrLeaseExpired        = errors.New("rsocket: lease expired")
	ErrLeaseNoMoreRequests = errors.New("rsocket: no more lease")
)

// Factory can be used to generate leases.
type Factory interface {
	// Next generate next lease chan.
	Next(ctx context.Context) (ch chan Lease, ok bool)
}

// Lease represents lease structure.
type Lease struct {
	TimeToLive       time.Duration
	NumberOfRequests uint32
	Metadata         []byte
}

// NewSimpleFactory creates a simple lease factory.
func NewSimpleFactory(interval, ttl, delay time.Duration, numberOfRequest uint32) (Factory, error) {
	if interval <= 0 {
		return nil, errors.Errorf("invalid simple lease interval: %s", interval)
	}
	if ttl <= 0 {
		return nil, errors.Errorf("invalid simple lease TTL: %s", ttl)
	}
	if numberOfRequest < 1 {
		return nil, errors.Errorf("invalid simple lease NumberOfRequest: %d", numberOfRequest)
	}
	return &simpleLease{
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

func (s *simpleLease) Next(ctx context.Context) (chan Lease, bool) {
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
		defer tk.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-tk.C:
				ch <- s.create()
			}
		}
	}(ctx, ch)
	return ch, true
}

func (s *simpleLease) create() Lease {
	return Lease{
		TimeToLive:       s.ttl,
		NumberOfRequests: s.n,
	}
}
