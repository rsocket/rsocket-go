package socket

import (
	"sync/atomic"
)

const (
	_maskStreamID uint64 = 0x7FFFFFFF
	_halfSeed     uint64 = 0x40000000
)

type StreamID interface {
	Next() (id uint32, lap1st bool)
}

type serverStreamIDs struct {
	cur uint64
}

func (p *serverStreamIDs) Next() (uint32, bool) {
	// 2,4,6,8...
	seed := atomic.AddUint64(&p.cur, 1)
	v := 2 * seed
	if v != 0 {
		return uint32(_maskStreamID & v), seed <= _halfSeed
	}
	return p.Next()
}

type clientStreamIDs struct {
	cur uint64
}

func (p *clientStreamIDs) Next() (uint32, bool) {
	// 1,3,5,7
	seed := atomic.AddUint64(&p.cur, 1)
	v := 2*(seed-1) + 1
	if v != 0 {
		return uint32(_maskStreamID & v), seed <= _halfSeed
	}
	return p.Next()
}
