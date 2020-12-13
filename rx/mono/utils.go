package mono

import (
	"context"

	"github.com/jjeffcaii/reactor-go"
	"github.com/jjeffcaii/reactor-go/mono"
	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

var empty = newProxy(mono.Empty())

func FromFunc(gen func(context.Context) (payload.Payload, error)) Mono {
	return Create(func(ctx context.Context, s Sink) {
		if v, err := gen(ctx); err != nil {
			s.Error(err)
		} else {
			s.Success(v)
		}
	})
}

// IsSubscribeAsync returns true if target Mono will be subscribed async.
func IsSubscribeAsync(m Mono) bool {
	return mono.IsSubscribeAsync(m.Raw())
}

// Raw wrap a low-level Mono.
func Raw(input mono.Mono) Mono {
	return newProxy(input)
}

func RawOneshot(origin mono.Mono) Mono {
	return borrowOneshotProxy(origin)
}

// Just wrap an exist Payload to a Mono.
func Just(input payload.Payload) Mono {
	return newProxy(mono.Just(input))
}

// JustOneshot wraps an existing Payload to an oneshot Mono.
func JustOneshot(input payload.Payload) Mono {
	return borrowOneshotProxy(mono.JustOneshot(input))
}

// JustOrEmpty wraps an existing Payload to a Mono.
// Payload could be nil here.
func JustOrEmpty(input payload.Payload) Mono {
	return newProxy(mono.JustOrEmpty(input))
}

// Empty returns an empty Mono.
func Empty() Mono {
	return empty
}

// Error wraps an error to a Mono.
func Error(err error) Mono {
	return newProxy(mono.Error(err))
}

// ErrorOneshot wraps an error to an oneshot Mono.
func ErrorOneshot(err error) Mono {
	return borrowOneshotProxy(mono.ErrorOneshot(err))
}

// Create wraps a generator function to a Mono.
func Create(gen func(context.Context, Sink)) Mono {
	return newProxy(mono.Create(func(ctx context.Context, sink mono.Sink) {
		gen(ctx, sinkProxy{sink})
	}))
}

// CreateOneshot wraps a generator function to an oneshot Mono.
func CreateOneshot(gen func(context.Context, Sink)) Mono {
	return borrowOneshotProxy(mono.CreateOneshot(func(ctx context.Context, sink mono.Sink) {
		gen(ctx, sinkProxy{sink})
	}))
}

func NewProcessor(sc scheduler.Scheduler, onFinally mono.ProcessorFinallyHook) (Mono, Sink, reactor.Disposable) {
	m, s, d := mono.NewProcessor(sc, onFinally)
	return borrowOneshotProxy(m), sinkProxy{native: s}, d
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

func subscribeWithChan(ctx context.Context, publisher mono.Mono, valueChan chan<- payload.Payload, errChan chan<- error, autoClose bool) {
	publisher.
		DoFinally(func(s reactor.SignalType) {
			if autoClose {
				defer close(valueChan)
				defer close(errChan)
			}
			if s == reactor.SignalTypeCancel {
				errChan <- reactor.ErrSubscribeCancelled
			}
		}).
		Subscribe(
			ctx,
			reactor.OnNext(func(v reactor.Any) error {
				valueChan <- v.(payload.Payload)
				return nil
			}),
			reactor.OnError(func(e error) {
				errChan <- e
			}),
		)
}

func toChan(ctx context.Context, publisher mono.Mono) (<-chan payload.Payload, <-chan error) {
	value := make(chan payload.Payload, 1)
	err := make(chan error, 1)
	subscribeWithChan(ctx, publisher, value, err, true)
	return value, err
}

func toBlock(ctx context.Context, m mono.Mono) (payload.Payload, error) {
	done := make(chan struct{})
	vchan := make(chan payload.Payload, 1)
	echan := make(chan error, 1)
	b := newBlockSubscriber(done, vchan, echan)
	m.SubscribeWith(ctx, b)
	<-done

	defer close(vchan)
	defer close(echan)

	select {
	case value := <-vchan:
		return value, nil
	case err := <-echan:
		return nil, err
	default:
		return nil, nil
	}
}

func unpackRawPublisher(source Mono) mono.Mono {
	if source == nil {
		return nil
	}
	switch t := source.(type) {
	case *oneshotProxy:
		return returnOneshotProxy(t)
	default:
		return t.Raw()
	}
}

func convertItem(item *reactor.Item) (result rx.Item, err error) {
	if item == nil {
		return
	}
	if item.E != nil {
		result.E = item.E
		return
	}
	if item.V == nil {
		return
	}
	p, ok := item.V.(payload.Payload)
	if !ok {
		err = errors.Errorf("require Payload value type instead of %t", item.V)
		return
	}
	result.V = p
	return
}
