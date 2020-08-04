package flux

import (
	"context"

	"github.com/jjeffcaii/reactor-go/flux"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

var empty = newProxy(flux.Empty())

// Raw creates a RSocket Flux from a native Flux in reactor-go.
// Don't use this API unless you know what you are doing.
func Raw(input flux.Flux) Flux {
	return newProxy(input)
}

// Empty returns a blank Flux.
func Empty() Flux {
	return empty
}

// FromSlice creates a Flux from a Payload slice.
func FromSlice(payloads []payload.Payload) Flux {
	return Just(payloads...)
}

// Just returns a Flux with some payloads.
func Just(payloads ...payload.Payload) Flux {
	totals := len(payloads)
	if totals < 1 {
		return newProxy(flux.Empty())
	}
	values := make([]interface{}, totals)
	for i := 0; i < totals; i++ {
		values[i] = payloads[i]
	}
	return newProxy(flux.Just(values...))
}

// Error returns a Flux with a custom error.
func Error(err error) Flux {
	return newProxy(flux.Error(err))
}

// Create creates a Flux by a generator func.
func Create(gen func(ctx context.Context, s Sink)) Flux {
	return newProxy(flux.Create(func(ctx context.Context, sink flux.Sink) {
		gen(ctx, newProxySink(sink))
	}))
}

// CreateProcessor creates a new Processor.
func CreateProcessor() Processor {
	p := flux.NewUnicastProcessor()
	return newProxy(p)
}

// Clone clones a Publisher to a Flux.
func Clone(source rx.Publisher) Flux {
	return Create(func(ctx context.Context, s Sink) {
		source.Subscribe(ctx,
			rx.OnNext(func(input payload.Payload) error {
				s.Next(input)
				return nil
			}),
			rx.OnComplete(func() {
				s.Complete()
			}),
			rx.OnError(func(e error) {
				s.Error(e)
			}),
		)
	})

}

// CreateFromChannel creates a Flux from channels.
func CreateFromChannel(payloads <-chan payload.Payload, err <-chan error) Flux {
	return Create(func(ctx context.Context, s Sink) {
		go func() {
		loop:
			for {
				select {
				case <-ctx.Done():
					s.Error(ctx.Err())
					break loop
				case p, ok := <-payloads:
					if ok {
						s.Next(p)
					} else {
						s.Complete()
						break loop
					}
				case e := <-err:
					if e != nil {
						s.Error(e)
						break loop
					}
				}
			}
		}()
	})
}
