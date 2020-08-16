package socket

import "sync"

type map32 struct {
	locker sync.RWMutex
	store  map[uint32]interface{}
}

func (p *map32) Destroy() {
	p.locker.Lock()
	p.store = nil
	p.locker.Unlock()
}

func (p *map32) Range(fn func(uint32, interface{}) bool) {
	p.locker.RLock()
	defer p.locker.RUnlock()
	for key, value := range p.store {
		if !fn(key, value) {
			break
		}
	}
}

func (p *map32) Load(key uint32) (v interface{}, ok bool) {
	p.locker.RLock()
	v, ok = p.store[key]
	p.locker.RUnlock()
	return
}

func (p *map32) Store(key uint32, value interface{}) {
	p.locker.Lock()
	if p.store != nil {
		p.store[key] = value
	}
	p.locker.Unlock()
}

func (p *map32) Delete(key uint32) {
	p.locker.Lock()
	delete(p.store, key)
	p.locker.Unlock()
}

func newMap32() *map32 {
	return &map32{
		store: make(map[uint32]interface{}),
	}
}
