package mono

import (
	"context"

	"github.com/jjeffcaii/reactor-go"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/payload"
)

type blockSubscriber struct {
	done  chan struct{}
	vchan chan<- payload.Payload
	echan chan<- error
}

func newBlockSubscriber(
	done chan struct{},
	vchan chan<- payload.Payload,
	echan chan<- error,
) reactor.Subscriber {
	return blockSubscriber{
		done:  done,
		vchan: vchan,
		echan: echan,
	}
}

func (b blockSubscriber) OnComplete() {
	select {
	case <-b.done:
	default:
		_ = common.SafeCloseDoneChan(b.done)
	}
}

func (b blockSubscriber) OnError(err error) {
	select {
	case <-b.done:
	default:
		if common.SafeCloseDoneChan(b.done) {
			b.echan <- err
		}
	}
}

func (b blockSubscriber) OnNext(any reactor.Any) {
	select {
	case <-b.done:
	default:
		if r, ok := any.(common.Releasable); ok {
			r.IncRef()
		}
		b.vchan <- any.(payload.Payload)
	}
}

func (b blockSubscriber) OnSubscribe(ctx context.Context, subscription reactor.Subscription) {
	// workaround: watch context
	if ctx != context.Background() && ctx != context.TODO() {
		go func() {
			select {
			case <-ctx.Done():
				b.OnError(reactor.ErrSubscribeCancelled)
			case <-b.done:
			}
		}()
	}
	subscription.Request(reactor.RequestInfinite)
}
