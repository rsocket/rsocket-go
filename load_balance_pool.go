package rsocket

import (
	"container/list"
	"errors"
)

var errNoSuchSocketSupplier = errors.New("no such socket supplier")

type socketSupplierPool struct {
	busy, idle *list.List
}

func (p *socketSupplierPool) Len() (n int) {
	n = p.idle.Len()
	return
}

func (p *socketSupplierPool) Next() (supplier *socketSupplier, ok bool) {
	size := p.idle.Len()
	if size < 1 {
		return
	}
	var found *list.Element
	for cur := p.idle.Front(); cur != nil; cur = cur.Next() {
		if found == nil {
			found = cur
		} else if cur.Value.(*socketSupplier).availability() > found.Value.(*socketSupplier).availability() {
			found = cur
		}
	}
	result := found.Value.(*socketSupplier)
	if result.availability() <= 0 {
		return
	}
	p.idle.Remove(found)
	p.busy.PushBack(result)

	supplier = result
	ok = true
	return
}

func (p *socketSupplierPool) returnSupplier(supplier *socketSupplier) error {
	var found *list.Element
	for found = p.busy.Front(); found != nil; found = found.Next() {
		if found.Value == supplier {
			break
		}
	}
	if found == nil {
		return errNoSuchSocketSupplier
	}
	p.busy.Remove(found)
	p.idle.PushBack(supplier)
	return nil
}

func (p *socketSupplierPool) borrowSupplier(supplier *socketSupplier) error {
	var found *list.Element
	for found = p.idle.Front(); found != nil; found = found.Next() {
		if found.Value == supplier {
			break
		}
	}
	if found == nil {
		return errNoSuchSocketSupplier
	}
	p.idle.Remove(found)
	p.busy.PushBack(supplier)
	return nil
}

func newSocketPool(first *socketSupplier, others ...*socketSupplier) (pool *socketSupplierPool) {
	pool = &socketSupplierPool{
		busy: list.New(),
		idle: list.New(),
	}
	pool.idle.PushBack(first)
	for _, it := range others {
		pool.idle.PushBack(it)
	}
	return
}
