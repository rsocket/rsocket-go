package flux

import (
	"context"
	"github.com/jjeffcaii/reactor-go/flux"
	"github.com/jjeffcaii/reactor-go/scheduler"
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

func Error(err error) Flux  {
	return Create(func(ctx context.Context, s Sink) {
		s.Error(err)
	})
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

func CreateFromChannel(payloads <-chan *payload.Payload, err <-chan error) Flux {
	flux := Create(func(ctx context.Context, s Sink) {
		worker := scheduler.Parallel().Worker()
		worker.Do(func() {
		loop:
			for {
				select {
				case p, o := <-payloads:
					if o {
						s.Next(*p)
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
		})
	})

	return flux
}

func ToChannel(input Flux, ctx context.Context) (<-chan *payload.Payload, <-chan error) {
	return ToChannelOnSchedulerWithSize(input, ctx, scheduler.Parallel(), 256)
}

func ToChannelWithSize(input Flux, ctx context.Context, size int32) (<-chan *payload.Payload, <-chan error) {
	return ToChannelOnSchedulerWithSize(input, ctx, scheduler.Parallel(), size)
}

func ToChannelOnScheduler(input Flux, ctx context.Context, scheduler scheduler.Scheduler) (<-chan *payload.Payload, <-chan error) {
	return ToChannelOnSchedulerWithSize(input, ctx, scheduler, 256)
}

func ToChannelOnSchedulerWithSize(input Flux, ctx context.Context, scheduler scheduler.Scheduler, size int32) (<-chan *payload.Payload, <-chan error) {
	errorChannel := make(chan error, 1)
	payloadChannel := make(chan *payload.Payload, size)

	input.
		SubscribeOn(scheduler).
		DoFinally(func(s rx.SignalType) {
			close(payloadChannel)
			close(errorChannel)
		}).
		Subscribe(ctx,
			rx.OnNext(
				func(input payload.Payload) {
					payloadChannel <- &input
				}), rx.OnError(func(e error) {
				errorChannel <- e
			}))

	return payloadChannel, errorChannel
}
