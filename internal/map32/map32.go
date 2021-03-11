package map32

import (
	"sync"
)

// Hasher calculates the sharding offset.
type Hasher func(key uint32, cap int) (offset int)

type shard32 struct {
	sync.Mutex
	store map[uint32]interface{}
}

func (s *shard32) Range(fn func(uint32, interface{}) bool) bool {
	if s == nil {
		return true
	}
	s.Lock()
	defer s.Unlock()

	if s.store == nil {
		return true
	}

	for k, v := range s.store {
		if !fn(k, v) {
			return false
		}
	}
	return true
}

func (s *shard32) Load(key uint32) (v interface{}, ok bool) {
	if s == nil {
		return
	}
	s.Lock()
	if s.store != nil {
		v, ok = s.store[key]
	}
	s.Unlock()
	return
}

func (s *shard32) Store(key uint32, value interface{}) {
	if s == nil {
		return
	}
	s.Lock()
	if s.store == nil {
		s.store = make(map[uint32]interface{})
	}
	s.store[key] = value
	s.Unlock()
}

func (s *shard32) Delete(key uint32) {
	if s == nil {
		return
	}
	s.Lock()
	if s.store != nil {
		delete(s.store, key)
	}
	s.Unlock()
}

type map32 struct {
	h   Hasher
	cap int
	s   []shard32
}

func (m *map32) Destroy() {
	m.cap = 0
	m.s = nil
}

func (m *map32) Range(fn func(uint32, interface{}) bool) {
	for i := 0; i < len(m.s); i++ {
		next := &m.s[i]
		if !next.Range(fn) {
			break
		}
	}
}

func (m *map32) Load(key uint32) (interface{}, bool) {
	return m.shard(key).Load(key)
}

func (m *map32) Store(key uint32, value interface{}) {
	m.shard(key).Store(key, value)
}

func (m *map32) Delete(key uint32) {
	m.shard(key).Delete(key)
}

func (m *map32) shard(k uint32) *shard32 {
	if m.cap == 0 {
		return nil
	}
	var offset int
	if m.h == nil {
		offset = int(k) % m.cap
	} else {
		offset = m.h(k, m.cap)
	}

	if offset < 0 || offset >= m.cap {
		return nil
	}
	return &m.s[offset]
}

type config struct {
	cap int
	h   Hasher
}

// Option configs the Map32.
type Option func(*config)

// WithCap sets cap for Map32.
func WithCap(cap int) Option {
	return func(c *config) {
		c.cap = cap
	}
}

// WithHasher sets Hasher for Map32.
func WithHasher(h Hasher) Option {
	return func(c *config) {
		c.h = h
	}
}

// New creates a new Map32 with input options.
func New(options ...Option) Map32 {
	c := config{
		cap: 32,
	}
	for _, it := range options {
		it(&c)
	}
	s := make([]shard32, c.cap)
	return &map32{
		h:   c.h,
		cap: c.cap,
		s:   s,
	}
}
