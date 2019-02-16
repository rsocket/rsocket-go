package rsocket

import (
	"context"
)

const (
	SigReady SignalType = iota
	SigSuccess
	SigError
	SigCancel
)

const (
	rxFuncFinally rxFuncType = iota
	rxFuncCancel
	rxFuncError
	rxFuncSuccess
	rxFuncClean
	rxFuncComplete
)

type SignalType uint8
type rxFuncType uint8

type RxFunc interface {
	RegisterError(fn OnError)
	RegisterFinally(fn OnFinally)
	RegisterCancel(fn OnCancel)
	RegisterNext(fn Consumer)
	RegisterSuccess(fn Consumer)
	RegisterAfterConsumer(fn Consumer)
	RegisterComplete(fn OnComplete)

	DoError(ctx context.Context, e error)
	DoCancel(ctx context.Context)
	DoFinally(ctx context.Context)
	DoNextOrSuccess(ctx context.Context, payload Payload)
	DoAfterConsumer(ctx context.Context, payload Payload)
	DoComplete(ctx context.Context)
}

type implRxFuncStore struct {
	values map[rxFuncType][]interface{}
}

func (p *implRxFuncStore) RegisterComplete(fn OnComplete) {
	p.register(rxFuncComplete, fn)
}

func (p *implRxFuncStore) DoComplete(ctx context.Context) {
	li, ok := p.values[rxFuncComplete]
	if !ok {
		return
	}
	for i, l := 0, len(li); i < l; i++ {
		li[i].(OnComplete)(ctx)
	}
}

func (p *implRxFuncStore) DoNextOrSuccess(ctx context.Context, payload Payload) {
	cur, ok := p.values[rxFuncSuccess]
	if !ok {
		return
	}
	for _, value := range cur {
		value.(Consumer)(ctx, payload)
	}
}

func (p *implRxFuncStore) DoAfterConsumer(ctx context.Context, payload Payload) {
	cur, ok := p.values[rxFuncClean]
	if !ok {
		return
	}
	for _, value := range cur {
		value.(Consumer)(ctx, payload)
	}
}

func (p *implRxFuncStore) DoFinally(ctx context.Context) {
	cur, ok := p.values[rxFuncFinally]
	if !ok {
		return
	}
	for i, l := 0, len(cur); i < l; i++ {
		cur[l-i-1].(OnFinally)(ctx)
	}
}

func (p *implRxFuncStore) DoCancel(ctx context.Context) {
	cur, ok := p.values[rxFuncCancel]
	if !ok {
		return
	}
	for _, value := range cur {
		value.(OnCancel)(ctx)
	}
}

func (p *implRxFuncStore) DoError(ctx context.Context, e error) {
	cur, ok := p.values[rxFuncError]
	if !ok {
		return
	}
	for _, value := range cur {
		value.(OnError)(ctx, e)
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
	p.values[key] = append(p.values[key], fn)
}

func newRxFuncStore() RxFunc {
	return &implRxFuncStore{
		values: make(map[rxFuncType][]interface{}, 6),
	}
}
