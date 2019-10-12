package mono

import (
	"context"

	"github.com/jjeffcaii/reactor-go/mono"
	"github.com/rsocket/rsocket-go/payload"
)

var empty = newProxy(mono.Empty())

func Raw(input mono.Mono) Mono {
	return newProxy(input)
}

func Just(input payload.Payload) Mono {
	return newProxy(mono.Just(input))
}

func JustOrEmpty(input payload.Payload) Mono {
	return newProxy(mono.JustOrEmpty(input))
}

func Empty() Mono {
	return empty
}

func Error(err error) Mono {
	return newProxy(mono.Error(err))
}

func Create(gen func(context.Context, Sink)) Mono {
	return newProxy(mono.Create(func(i context.Context, sink mono.Sink) {
		gen(i, sinkProxy{sink})
	}))
}

func CreateProcessor() Processor {
	return newProxy(mono.CreateProcessor())
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
