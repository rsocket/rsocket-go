package mono

import (
	"context"

	"github.com/jjeffcaii/reactor-go/mono"
	"github.com/rsocket/rsocket-go/payload"
)

var empty = newProxy(mono.Empty())

// Raw wrap a low-level Mono.
func Raw(input mono.Mono) Mono {
	return getObjFromPool(input)
}

// Just wrap an exist Payload to a Mono.
func Just(input payload.Payload) Mono {
	return getObjFromPool(mono.Just(input))
}

// JustOrEmpty wrap an exist Payload to Mono.
// Payload could be nil here.
func JustOrEmpty(input payload.Payload) Mono {
	return getObjFromPool(mono.JustOrEmpty(input))
}

// Empty returns an empty Mono.
func Empty() Mono {
	return empty
}

// Error wrap an error to a Mono.
func Error(err error) Mono {
	return getObjFromPool(mono.Error(err))
}

// Create wrap a generator function to a Mono.
func Create(gen func(context.Context, Sink)) Mono {
	return getObjFromPool(mono.Create(func(i context.Context, sink mono.Sink) {
		gen(i, sinkProxy{sink})
	}))
}

// CreateProcessor creates a Processor.
func CreateProcessor() Processor {
	return getObjFromPool(mono.CreateProcessor())
}

type sinkProxy struct {
	native mono.Sink
}

func (s sinkProxy) Success(in payload.Payload) {
	s.native.Success(in)
}

func (s sinkProxy) Error(e error) {
	s.native.Error(e)
}

// CreateFromChannel creates a Mono from channels.
func CreateFromChannel(payloads <-chan payload.Payload, err <-chan error) Mono {
	return Create(func(ctx context.Context, s Sink) {
		go func() {
		loop:
			for {
				select {
				case <-ctx.Done():
					if e := ctx.Err(); e != nil {
						s.Error(e)
					} else {
						s.Success(nil)
					}
					break loop
				case p, ok := <-payloads:
					if ok {
						s.Success(p)
					} else {
						s.Success(nil)
					}
					break loop
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
