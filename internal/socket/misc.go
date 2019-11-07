package socket

import (
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/framing"
	"github.com/rsocket/rsocket-go/rx"
)

type u32map struct {
	k sync.RWMutex
	m map[uint32]interface{}
}

func (p *u32map) Close() error {
	p.k.Lock()
	p.m = nil
	p.k.Unlock()
	return nil
}

func (p *u32map) Range(fn func(uint32, interface{}) bool) {
	p.k.RLock()
	for key, value := range p.m {
		if !fn(key, value) {
			break
		}
	}
	p.k.RUnlock()
}

func (p *u32map) Load(key uint32) (v interface{}, ok bool) {
	p.k.RLock()
	v, ok = p.m[key]
	p.k.RUnlock()
	return
}

func (p *u32map) Store(key uint32, value interface{}) {
	p.k.Lock()
	if p.m != nil {
		p.m[key] = value
	}
	p.k.Unlock()
}

func (p *u32map) Delete(key uint32) {
	p.k.Lock()
	delete(p.m, key)
	p.k.Unlock()
}

func newU32Map() *u32map {
	return &u32map{
		m: make(map[uint32]interface{}),
	}
}

// SetupInfo represents basic info of setup.
type SetupInfo struct {
	Lease             bool
	Version           common.Version
	KeepaliveInterval time.Duration
	KeepaliveLifetime time.Duration
	Token             []byte
	DataMimeType      []byte
	Data              []byte
	MetadataMimeType  []byte
	Metadata          []byte
}

func (p *SetupInfo) toFrame() *framing.FrameSetup {
	return framing.NewFrameSetup(
		p.Version,
		p.KeepaliveInterval,
		p.KeepaliveLifetime,
		p.Token,
		p.MetadataMimeType,
		p.DataMimeType,
		p.Data,
		p.Metadata,
		p.Lease,
	)
}

func tryRecover(e interface{}) (err error) {
	if e == nil {
		return
	}
	switch v := e.(type) {
	case error:
		err = v
	case string:
		err = errors.New(v)
	default:
		err = errors.Errorf("error: %s", v)
	}
	return
}

func toIntN(n uint32) int {
	if n > rx.RequestMax {
		return rx.RequestMax
	}
	return int(n)
}

func toU32N(n int) uint32 {
	if n > rx.RequestMax {
		return rx.RequestMax
	}
	return uint32(n)
}
