package common

import (
	"sync"
)

const _slots = 2 * 2 * 2 * 2

type U32Map interface {
	Clear()
	Range(fn func(k uint32, v interface{}) bool)
	Load(key uint32) (v interface{}, ok bool)
	Store(key uint32, value interface{})
	Delete(key uint32)
}

type u32map struct {
	slots [_slots]*u32slot
}

func (u *u32map) Clear() {
	for _, slot := range u.slots {
		slot.Clear()
	}
}

func (u *u32map) Range(fn func(k uint32, v interface{}) bool) {
	for _, slot := range u.slots {
		if !slot.innerRange(fn) {
			return
		}
	}
}

func (u *u32map) Load(key uint32) (v interface{}, ok bool) {
	return u.seek(key).Load(key)
}

func (u *u32map) Store(key uint32, value interface{}) {
	u.seek(key).Store(key, value)
}

func (u *u32map) Delete(key uint32) {
	u.seek(key).Delete(key)
}

func (u *u32map) seek(key uint32) *u32slot {
	k := key & (_slots - 1)
	return u.slots[k]
}

type u32slot struct {
	k sync.RWMutex
	m map[uint32]interface{}
}

func (u *u32slot) Clear() {
	if u == nil || u.m == nil {
		return
	}
	u.k.Lock()
	u.m = nil
	u.k.Unlock()
}

func (u *u32slot) Range(fn func(k uint32, v interface{}) bool) {
	u.innerRange(fn)
}

func (u *u32slot) Load(key uint32) (v interface{}, ok bool) {
	if u == nil || u.m == nil {
		return
	}
	u.k.RLock()
	v, ok = u.m[key]
	u.k.RUnlock()
	return
}

func (u *u32slot) Store(key uint32, value interface{}) {
	if u == nil || u.m == nil {
		return
	}
	u.k.Lock()
	u.m[key] = value
	u.k.Unlock()
}

func (u *u32slot) Delete(key uint32) {
	if u == nil || u.m == nil {
		return
	}
	u.k.Lock()
	delete(u.m, key)
	u.k.Unlock()
}

func (u *u32slot) innerRange(fn func(k uint32, v interface{}) bool) bool {
	if u == nil || u.m == nil {
		return false
	}
	u.k.RLock()
	defer u.k.RUnlock()
	for key, value := range u.m {
		if !fn(key, value) {
			return false
		}
	}
	return true
}

func NewU32MapLite() U32Map {
	return &u32slot{
		m: make(map[uint32]interface{}),
	}
}

func NewU32Map() U32Map {
	var slots [_slots]*u32slot
	for i := 0; i < len(slots); i++ {
		slots[i] = &u32slot{
			m: make(map[uint32]interface{}),
		}
	}
	return &u32map{slots: slots}
}
