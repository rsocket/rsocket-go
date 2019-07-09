package flux

import (
	"context"

	"github.com/jjeffcaii/reactor-go/flux"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

var empty = newProxy(flux.Empty())

func Raw(input flux.Flux) Flux {
	return newProxy(input)
}

func Empty() Flux {
	return empty
}

func Just(payloads ...payload.Payload) Flux {
	totals := len(payloads)
	if totals < 1 {
		return newProxy(flux.Just())
	}
	values := make([]interface{}, totals)
	for i := 0; i < totals; i++ {
		values[i] = payloads[i]
	}
	return newProxy(flux.Just(values...))
}

func Create(gen func(ctx context.Context, s Sink)) Flux {
	return newProxy(flux.Create(func(ctx context.Context, sink flux.Sink) {
		gen(ctx, newProxySink(sink))
	}))
}

func CreateProcessor() Processor {
	proc := flux.NewUnicastProcessor()
	return newProxy(proc)
}

func Clone(source rx.Publisher) Flux {
	return Create(func(ctx context.Context, s Sink) {
		source.Subscribe(ctx, rx.OnNext(func(input payload.Payload) {
			s.Next(input)
		}), rx.OnComplete(func() {
			s.Complete()
		}), rx.OnError(func(e error) {
			s.Error(e)
		}))
	})

}
