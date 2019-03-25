package rx

import (
	"context"
	"github.com/rsocket/rsocket-go/payload"
)

// NewFluxFromArray returns a new Flux with payloads.
func NewFluxFromArray(first payload.Payload, others ...payload.Payload) Flux {
	// TODO: tuning
	return NewFlux(func(ctx context.Context, producer Producer) {
		_ = producer.Next(first)
		for _, it := range others {
			_ = producer.Next(it)
		}
		producer.Complete()
	})
}
