package socket

import "sync"

type map32 struct {
	sync.RWMutex
	store map[uint32]interface{}
}

func (p *map32) Destroy() {
	p.Lock()
	p.store = nil
	p.Unlock()
}

func (p *map32) Range(fn func(uint32, interface{}) bool) {
	p.RLock()
	defer p.RUnlock()
	for key, value := range p.store {
		if !fn(key, value) {
			break
		}
	}
}

func (p *map32) Load(key uint32) (v interface{}, ok bool) {
	p.RLock()
	v, ok = p.store[key]
	p.RUnlock()
	return
}

func (p *map32) Store(key uint32, value interface{}) {
	p.Lock()
	if p.store != nil {
		p.store[key] = value
	}
	p.Unlock()
}

func (p *map32) Delete(key uint32) {
	p.Lock()
	delete(p.store, key)
	p.Unlock()
}

func newMap32() *map32 {
	return &map32{
		store: make(map[uint32]interface{}),
	}
}
