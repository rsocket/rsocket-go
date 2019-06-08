package rx

import (
	"context"
	"errors"
	"fmt"

	"github.com/rsocket/rsocket-go/internal/logger"
	"github.com/rsocket/rsocket-go/payload"
)

// ToFlux converts Publisher to Flux.
func ToFlux(publisher Publisher) Flux {
	if v, ok := publisher.(Flux); ok {
		return v
	}
	switch v := publisher.(type) {
	case Flux:
		return v
	case Mono:
		flux := NewFlux(nil)
		v.SubscribeOn(ElasticScheduler()).Subscribe(context.Background(),
			OnNext(func(ctx context.Context, s Subscription, elem payload.Payload) {
				if err := flux.(Producer).Next(elem); err != nil {
					logger.Warnf("emit next element failed: %s\n", err)
				}
				flux.(Producer).Complete()
			}),
			OnError(func(ctx context.Context, err error) {
				flux.(Producer).Error(err)
			}),
		)
		return flux
	}
	panic("unreachable")
}

// ToMono converts Publisher to Mono.
func ToMono(publisher Publisher) Mono {
	switch v := publisher.(type) {
	case Mono:
		return v
	case Flux:
		return NewMono(func(ctx context.Context, sink MonoProducer) {
			v.
				DoOnNext(func(ctx context.Context, s Subscription, elem payload.Payload) {
					if !v.(Disposable).IsDisposed() {
						_ = sink.Success(elem)
						s.Cancel()
					}
				}).
				DoOnError(func(ctx context.Context, err error) {
					sink.Error(err)
				}).
				Subscribe(context.Background())
		})
	}
	panic("unreachable")
}

// IntRange is utilities for range operations.
type IntRange struct {
	from, to int
}

// Map converts int to Payload.
func (p *IntRange) Map(fn func(n int) payload.Payload) Flux {
	return NewFlux(func(ctx context.Context, producer Producer) {
		defer func() {
			e := recover()
			if e == nil {
				return
			}
			switch v := e.(type) {
			case error:
				producer.Error(v)
			case string:
				producer.Error(errors.New(v))
			default:
				producer.Error(fmt.Errorf("%s", v))
			}
		}()
		for i := p.from; i < p.to; i++ {
			err := producer.Next(fn(i))
			if err != nil {
				break
			}
		}
		producer.Complete()
	})
}

// Range returns a IntRange.
func Range(from, to int) *IntRange {
	return &IntRange{from: from, to: to}
}
