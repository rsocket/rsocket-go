package rsocket

import (
	"container/list"
)

type socketSupplierPool struct {
	busy, idle *list.List
}

func (p *socketSupplierPool) size() (n int) {
	n = p.idle.Len()
	return
}

func (p *socketSupplierPool) next() (supplier *socketSupplier, ok bool) {
	size := p.idle.Len()
	if size < 1 {
		return
	}

	var a, b *list.Element
	for cur := p.idle.Front(); cur != nil; cur = cur.Next() {
		// skip unhealth socket supplier
		if cur.Value.(*socketSupplier).availability() <= 0 {
			continue
		}
		if a != nil {
			b = cur
			break
		}
		a = cur
	}

	if a == nil {
		return
	}
	var choose *list.Element
	if b == nil {
		choose = a
	} else if a.Value.(*socketSupplier).availability() > b.Value.(*socketSupplier).availability() {
		choose = a
	} else {
		choose = b
	}
	supplier = choose.Value.(*socketSupplier)
	ok = true
	p.idle.Remove(choose)
	p.busy.PushBack(supplier)
	return
}

func (p *socketSupplierPool) returnSupplier(supplier *socketSupplier) (ok bool) {
	var found *list.Element
	for found = p.busy.Front(); found != nil; found = found.Next() {
		if found.Value == supplier {
			break
		}
	}
	if found != nil {
		p.busy.Remove(found)
		p.idle.PushBack(supplier)
		ok = true
	}
	return
}

func (p *socketSupplierPool) borrowSupplier(supplier *socketSupplier) (ok bool) {
	var found *list.Element
	for found = p.idle.Front(); found != nil; found = found.Next() {
		if found.Value == supplier {
			break
		}
	}
	if found != nil {
		p.idle.Remove(found)
		p.busy.PushBack(supplier)
		ok = true
	}
	return
}

type taged struct {
	elem *list.Element
	drop bool
}

func (p *socketSupplierPool) reset(newest map[string]*socketSupplier) (remove int, append int) {
	mBusy := make(map[string]taged)
	for cur := p.busy.Front(); cur != nil; cur = cur.Next() {
		v := cur.Value.(*socketSupplier)
		_, ok := newest[v.u]
		mBusy[v.u] = taged{cur, !ok}
	}
	mIdle := make(map[string]taged)
	for cur := p.idle.Front(); cur != nil; cur = cur.Next() {
		v := cur.Value.(*socketSupplier)
		_, ok := newest[v.u]
		mIdle[v.u] = taged{cur, !ok}
	}
	for _, v := range mBusy {
		if v.drop {
			p.busy.Remove(v.elem)
			remove++
		}
	}
	for _, v := range mIdle {
		if v.drop {
			p.idle.Remove(v.elem)
			remove++
		}
	}
	for k, v := range newest {
		var ok bool
		_, ok = mBusy[k]
		if ok {
			continue
		}
		_, ok = mIdle[k]
		if ok {
			continue
		}
		p.idle.PushBack(v)
		append++
	}
	return
}

func newSocketPool(suppliers ...*socketSupplier) (pool *socketSupplierPool) {
	pool = &socketSupplierPool{
		busy: list.New(),
		idle: list.New(),
	}
	for _, it := range suppliers {
		pool.idle.PushBack(it)
	}
	return
}
