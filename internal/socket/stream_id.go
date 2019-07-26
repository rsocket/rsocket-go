package socket

import (
	"sync/atomic"
)

const (
	maskStreamID uint64 = 0x7FFFFFFF
	halfSeed     uint64 = 0x40000000
)

type streamIDs interface {
	next() (id uint32, lap1st bool)
}

type serverStreamIDs struct {
	cur uint64
}

func (p *serverStreamIDs) next() (uint32, bool) {
	// 2,4,6,8...
	seed := atomic.AddUint64(&p.cur, 1)
	v := 2 * seed
	if v != 0 {
		return uint32(maskStreamID & v), seed <= halfSeed
	}
	return p.next()
}

type clientStreamIDs struct {
	cur uint64
}

func (p *clientStreamIDs) next() (uint32, bool) {
	// 1,3,5,7
	seed := atomic.AddUint64(&p.cur, 1)
	v := 2*(seed-1) + 1
	if v != 0 {
		return uint32(maskStreamID & v), seed <= halfSeed
	}
	return p.next()
}
