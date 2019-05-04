package socket

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/framing"
	"github.com/rsocket/rsocket-go/rx"
)

type msgStoreMode int8

const (
	msgStoreModeRequestResponse msgStoreMode = iota
	msgStoreModeRequestStream
	msgStoreModeRequestChannel
)

var publishersPool = sync.Pool{
	New: func() interface{} {
		return &publishers{}
	},
}

type SetupInfo struct {
	Version           common.Version
	KeepaliveInterval time.Duration
	KeepaliveLifetime time.Duration
	Token             []byte
	DataMimeType      []byte
	Data              []byte
	MetadataMimeType  []byte
	Metadata          []byte
}

func (p *SetupInfo) ToFrame() *framing.FrameSetup {
	return framing.NewFrameSetup(
		p.Version,
		p.KeepaliveInterval,
		p.KeepaliveLifetime,
		p.Token,
		p.MetadataMimeType,
		p.DataMimeType,
		p.Data,
		p.Metadata,
	)
}

func borrowPublishers(mode msgStoreMode, sending, receiving rx.Publisher) (b *publishers) {
	b = publishersPool.Get().(*publishers)
	b.mode = mode
	b.sending = sending
	b.receiving = receiving
	return
}

func returnPublishers(b *publishers) {
	b.receiving = nil
	b.sending = nil
	publishersPool.Put(b)
}

type publishers struct {
	mode               msgStoreMode
	sending, receiving rx.Publisher
}

type publishersMap struct {
	mutex *sync.RWMutex
	m     map[uint32]*publishers
}

func (p *publishersMap) each(fn func(id uint32, elem *publishers)) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	for k, v := range p.m {
		fn(k, v)
	}
}

func (p *publishersMap) size() (n int) {
	p.mutex.RLock()
	n = len(p.m)
	p.mutex.RUnlock()
	return
}

func (p *publishersMap) put(id uint32, mode msgStoreMode, sending, receiving rx.Publisher) {
	p.mutex.Lock()
	p.m[id] = borrowPublishers(mode, sending, receiving)
	p.mutex.Unlock()
}

func (p *publishersMap) load(id uint32) (v *publishers, ok bool) {
	p.mutex.RLock()
	v, ok = p.m[id]
	p.mutex.RUnlock()
	return
}

func (p *publishersMap) remove(id uint32) {
	p.mutex.Lock()
	found, ok := p.m[id]
	if ok {
		delete(p.m, id)
		p.mutex.Unlock()
		returnPublishers(found)
	} else {
		p.mutex.Unlock()
	}
}

func newMessageStore() *publishersMap {
	return &publishersMap{
		mutex: &sync.RWMutex{},
		m:     make(map[uint32]*publishers),
	}
}

type streamIDs interface {
	next() uint32
}

type serverStreamIDs struct {
	cur uint32
}

func (p *serverStreamIDs) next() uint32 {
	var v uint32
	// 2,4,6,8...
	v = 2 * atomic.AddUint32(&p.cur, 1)
	if v != 0 {
		return v
	}
	return p.next()
}

type clientStreamIDs struct {
	cur uint32
}

func (p *clientStreamIDs) next() uint32 {
	var v uint32
	// 1,3,5,7
	v = 2*(atomic.AddUint32(&p.cur, 1)-1) + 1
	if v != 0 {
		return v
	}
	return p.next()
}

type errorProducer interface {
	Error(err error)
}

// toError try convert something to error
func toError(err interface{}) error {
	if err == nil {
		return nil
	}
	switch v := err.(type) {
	case error:
		return v
	case string:
		return errors.New(v)
	default:
		return fmt.Errorf("%s", v)
	}
}
