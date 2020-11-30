package mono

import (
	"context"
	"math"
	"sync"
	"sync/atomic"

	"github.com/jjeffcaii/reactor-go"
	"github.com/jjeffcaii/reactor-go/hooks"
	"github.com/rsocket/rsocket-go/internal/common"
)

var globalBlockSubscriberPool blockSubscriberPool

type blockSubscriberPool struct {
	inner sync.Pool
}

func (bp *blockSubscriberPool) get() *blockSubscriber {
	if exist, _ := bp.inner.Get().(*blockSubscriber); exist != nil {
		atomic.StoreInt32(&exist.done, 0)
		return exist
	}
	return &blockSubscriber{
		doneChan: make(chan struct{}, 1),
	}
}

func (bp *blockSubscriberPool) put(s *blockSubscriber) {
	if s == nil {
		return
	}
	s.Reset()
	bp.inner.Put(s)
}

var _ reactor.Subscriber = (*blockSubscriber)(nil)

type blockSubscriber struct {
	reactor.Item
	doneChan chan struct{}
	ctxChan  chan struct{}
	done     int32
}

func (b *blockSubscriber) Reset() {
	b.V = nil
	b.E = nil
	b.ctxChan = nil
	atomic.StoreInt32(&b.done, math.MinInt32)
}

func (b *blockSubscriber) Done() <-chan struct{} {
	return b.doneChan
}

func (b *blockSubscriber) OnComplete() {
	if atomic.CompareAndSwapInt32(&b.done, 0, 1) {
		b.finish()
	}
}

func (b *blockSubscriber) OnError(err error) {
	if !atomic.CompareAndSwapInt32(&b.done, 0, 1) {
		hooks.Global().OnErrorDrop(err)
		return
	}
	b.E = err
	b.finish()
}

func (b *blockSubscriber) finish() {
	if b.ctxChan != nil {
		close(b.ctxChan)
	}
	b.doneChan <- struct{}{}
}

func (b *blockSubscriber) OnNext(any reactor.Any) {
	if atomic.LoadInt32(&b.done) != 0 || b.V != nil || b.E != nil {
		hooks.Global().OnNextDrop(any)
		return
	}
	if r, _ := any.(common.Releasable); r != nil {
		r.IncRef()
	}
	b.V = any
}

func (b *blockSubscriber) OnSubscribe(ctx context.Context, su reactor.Subscription) {
	// workaround: watch context
	if ctx != context.Background() && ctx != context.TODO() {
		ctxChan := make(chan struct{})
		b.ctxChan = ctxChan
		go func() {
			select {
			case <-ctx.Done():
				b.OnError(reactor.NewContextError(ctx.Err()))
			case <-ctxChan:
			}
		}()
	}
	su.Request(reactor.RequestInfinite)
}
