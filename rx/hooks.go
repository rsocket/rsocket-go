package rx

import (
	"context"
	"sync"

	"github.com/rsocket/rsocket-go/payload"
)

var hooksPool = sync.Pool{
	New: func() interface{} {
		return &hooks{}
	},
}

// hooks is used to handle rx lifecycle events.
type hooks struct {
	hOnCancel    []FnOnCancel
	hOnRequest   []FnOnRequest
	hOnSubscribe []FnOnSubscribe
	hOnNext      []FnOnNext
	hOnComplete  []FnOnComplete
	hOnError     []FnOnError
	hOnFinally   []FnOnFinally
	hOnAfterNext []FnConsumer
}

func (p *hooks) Reset() {
	if p.hOnCancel != nil {
		p.hOnCancel = p.hOnCancel[:0]
	}
	if p.hOnRequest != nil {
		p.hOnRequest = p.hOnRequest[:0]
	}
	if p.hOnSubscribe != nil {
		p.hOnSubscribe = p.hOnSubscribe[:0]
	}
	if p.hOnNext != nil {
		p.hOnNext = p.hOnNext[:0]
	}
	if p.hOnComplete != nil {
		p.hOnComplete = p.hOnComplete[:0]
	}
	if p.hOnError != nil {
		p.hOnError = p.hOnError[:0]
	}
	if p.hOnFinally != nil {
		p.hOnFinally = p.hOnFinally[:0]
	}
	if p.hOnAfterNext != nil {
		p.hOnAfterNext = p.hOnAfterNext[:0]
	}
}

func (p *hooks) OnCancel(ctx context.Context) {
	if p == nil {
		return
	}
	for i, l := 0, len(p.hOnCancel); i < l; i++ {
		p.hOnCancel[i](ctx)
	}
}

func (p *hooks) OnRequest(ctx context.Context, n int) {
	if p == nil {
		return
	}
	for i, l := 0, len(p.hOnRequest); i < l; i++ {
		p.hOnRequest[i](ctx, n)
	}
}

func (p *hooks) OnSubscribe(ctx context.Context, s Subscription) {
	if p == nil {
		return
	}
	for i, l := 0, len(p.hOnSubscribe); i < l; i++ {
		p.hOnSubscribe[l-i-1](ctx, s)
	}
}

func (p *hooks) OnNext(ctx context.Context, s Subscription, elem payload.Payload) {
	if p == nil {
		return
	}
	for i, l := 0, len(p.hOnNext); i < l; i++ {
		p.hOnNext[i](ctx, s, elem)
	}
}

func (p *hooks) OnComplete(ctx context.Context) {
	if p == nil {
		return
	}
	for i, l := 0, len(p.hOnComplete); i < l; i++ {
		p.hOnComplete[i](ctx)
	}
}

func (p *hooks) OnError(ctx context.Context, err error) {
	if p == nil {
		return
	}
	for i, l := 0, len(p.hOnError); i < l; i++ {
		p.hOnError[i](ctx, err)
	}
}

func (p *hooks) OnFinally(ctx context.Context, sig SignalType) {
	if p == nil {
		return
	}
	for i, l := 0, len(p.hOnFinally); i < l; i++ {
		p.hOnFinally[l-i-1](ctx, sig)
	}
}

func (p *hooks) OnAfterNext(ctx context.Context, elem payload.Payload) {
	if p == nil {
		return
	}
	for i, l := 0, len(p.hOnAfterNext); i < l; i++ {
		p.hOnAfterNext[i](ctx, elem)
	}
}

func (p *hooks) DoOnAfterNext(fn FnConsumer) {
	p.hOnAfterNext = append(p.hOnAfterNext, fn)
}

func (p *hooks) DoOnError(fn FnOnError) {
	p.hOnError = append(p.hOnError, fn)
}

func (p *hooks) DoOnNext(fn FnOnNext) {
	p.hOnNext = append(p.hOnNext, fn)
}

func (p *hooks) DoOnRequest(fn FnOnRequest) {
	p.hOnRequest = append(p.hOnRequest, fn)
}

func (p *hooks) DoOnComplete(fn FnOnComplete) {
	p.hOnComplete = append(p.hOnComplete, fn)
}

func (p *hooks) DoOnCancel(fn FnOnCancel) {
	p.hOnCancel = append(p.hOnCancel, fn)
}

func (p *hooks) DoOnSubscribe(fn FnOnSubscribe) {
	p.hOnSubscribe = append(p.hOnSubscribe, fn)
}
func (p *hooks) DoOnFinally(fn FnOnFinally) {
	p.hOnFinally = append(p.hOnFinally, fn)
}

func borrowHooks() *hooks {
	return hooksPool.Get().(*hooks)
}
func returnHooks(h *hooks) {
	h.Reset()
	hooksPool.Put(h)
}
