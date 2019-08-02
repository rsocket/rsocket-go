package mono

import (
	"context"

	"github.com/jjeffcaii/reactor-go/mono"
	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
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
	return Create(func(i context.Context, sink Sink) {
		sink.Error(err)
	})
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

func CreateFromChannel(payloads <-chan payload.Payload, err <-chan error) Mono {
	mono := Create(func(ctx context.Context, s Sink) {
		worker := scheduler.Parallel().Worker()
		worker.Do(func() {
		loop:
			for {
				select {
				case p, o := <-payloads:
					if o {
						s.Success(p)
						break loop
					} else {
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

	return mono
}

func ToChannel(input Mono, ctx context.Context) (<-chan payload.Payload, <-chan error) {
	return ToChannelOnScheduler(input, ctx, scheduler.Parallel())
}

func ToChannelOnScheduler(input Mono, ctx context.Context, scheduler scheduler.Scheduler) (<-chan payload.Payload, <-chan error) {
	errorChannel := make(chan error, 1)
	payloadChannel := make(chan payload.Payload, 1)

	input.SubscribeOn(scheduler).DoOnSuccess(func(input payload.Payload) {
		payloadChannel <- input
	}).
		DoOnError(func(e error) {
			errorChannel <- e
		}).
		DoFinally(func(s rx.SignalType) {
			close(payloadChannel)
			close(errorChannel)
		}).Subscribe(ctx)

	return payloadChannel, errorChannel
}
