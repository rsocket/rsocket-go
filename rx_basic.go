package rsocket

import (
	"container/list"
	"context"
)

const (
	rxFuncFinally rxFuncType = iota
	rxFuncCancel
	rxFuncError
	rxFuncSuccess
	rxFuncClean
	rxFuncComplete
	rxFuncDrain
	rxFuncSub
)

type rxFuncType uint8

type RxFunc interface {
	RegisterError(fn OnError)
	RegisterFinally(fn OnFinally)
	RegisterCancel(fn OnCancel)
	RegisterNext(fn Consumer)
	RegisterSuccess(fn Consumer)
	RegisterAfterConsumer(fn Consumer)
	RegisterComplete(fn OnComplete)
	RegisterDrain(fn OnExhaust)
	RegisterSubscribe(fn OnSubscribe)

	DoError(ctx context.Context, e error)
	DoCancel(ctx context.Context)
	DoFinally(ctx context.Context, sig SignalType)
	DoNextOrSuccess(ctx context.Context, payload Payload)
	DoAfterConsumer(ctx context.Context, payload Payload)
	DoComplete(ctx context.Context)
	DoDrain(ctx context.Context)
	DoSubscribe(ctx context.Context)
}

type implRxFuncStore struct {
	values [8]*list.List
}

func (p *implRxFuncStore) RegisterSubscribe(fn OnSubscribe) {
	p.register(rxFuncSub, fn)
}

func (p *implRxFuncStore) RegisterDrain(fn OnExhaust) {
	p.register(rxFuncDrain, fn)
}

func (p *implRxFuncStore) RegisterComplete(fn OnComplete) {
	p.register(rxFuncComplete, fn)
}

func (p *implRxFuncStore) DoComplete(ctx context.Context) {
	root := p.values[rxFuncComplete]
	if root == nil {
		return
	}
	var next *list.Element
	for it := root.Front(); it != nil; it = next {
		it.Value.(OnComplete)(ctx)
		next = it.Next()
	}
}

func (p *implRxFuncStore) DoNextOrSuccess(ctx context.Context, payload Payload) {
	root := p.values[rxFuncSuccess]
	if root == nil {
		return
	}
	var next *list.Element
	for it := root.Front(); it != nil; it = next {
		it.Value.(Consumer)(ctx, payload)
		next = it.Next()
	}
}

func (p *implRxFuncStore) DoAfterConsumer(ctx context.Context, payload Payload) {
	root := p.values[rxFuncClean]
	if root == nil {
		return
	}
	var next *list.Element
	for it := root.Front(); it != nil; it = next {
		it.Value.(Consumer)(ctx, payload)
		next = it.Next()
	}
}

func (p *implRxFuncStore) DoFinally(ctx context.Context, sig SignalType) {
	root := p.values[rxFuncFinally]
	if root == nil {
		return
	}
	var prev *list.Element
	for it := root.Back(); it != nil; it = prev {
		it.Value.(OnFinally)(ctx, sig)
		prev = it.Prev()
	}
}

func (p *implRxFuncStore) DoSubscribe(ctx context.Context) {
	root := p.values[rxFuncSub]
	if root == nil {
		return
	}
	var next *list.Element
	for it := root.Front(); it != nil; it = next {
		it.Value.(OnSubscribe)(ctx)
		next = it.Next()
	}
}

func (p *implRxFuncStore) DoCancel(ctx context.Context) {
	root := p.values[rxFuncCancel]
	if root == nil {
		return
	}
	var next *list.Element
	for it := root.Front(); it != nil; it = next {
		it.Value.(OnCancel)(ctx)
		next = it.Next()
	}
}

func (p *implRxFuncStore) DoError(ctx context.Context, e error) {
	li := p.values[rxFuncError]
	if li == nil {
		return
	}
	var next *list.Element
	for it := li.Front(); it != nil; it = next {
		it.Value.(OnError)(ctx, e)
		next = it.Next()
	}
}

func (p *implRxFuncStore) DoDrain(ctx context.Context) {
	root := p.values[rxFuncDrain]
	if root == nil {
		return
	}
	var next *list.Element
	for it := root.Front(); it != nil; it = next {
		it.Value.(OnExhaust)(ctx)
		next = it.Next()
	}
}

func (p *implRxFuncStore) RegisterAfterConsumer(fn Consumer) {
	p.register(rxFuncClean, fn)
}

func (p *implRxFuncStore) RegisterFinally(fn OnFinally) {
	p.register(rxFuncFinally, fn)
}

func (p *implRxFuncStore) RegisterCancel(fn OnCancel) {
	p.register(rxFuncCancel, fn)
}

func (p *implRxFuncStore) RegisterNext(fn Consumer) {
	p.RegisterSuccess(func(ctx context.Context, item Payload) {
		if item != nil {
			fn(ctx, item)
		}
	})
}

func (p *implRxFuncStore) RegisterSuccess(fn Consumer) {
	p.register(rxFuncSuccess, fn)
}

func (p *implRxFuncStore) RegisterError(fn OnError) {
	p.register(rxFuncError, fn)
}

func (p *implRxFuncStore) register(key rxFuncType, fn interface{}) {
	root := p.values[key]
	if root == nil {
		root = list.New()
		p.values[key] = root
	}
	root.PushBack(fn)
}

func newRxFuncStore() RxFunc {
	return &implRxFuncStore{}
}
