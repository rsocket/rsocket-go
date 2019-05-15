package rsocket

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/rsocket/rsocket-go/rx"
)

type msgStoreMode int8

const (
	msgStoreModeRequestResponse msgStoreMode = iota
	msgStoreModeRequestStream
	msgStoreModeRequestChannel
)

type publishers struct {
	mode               msgStoreMode
	sending, receiving rx.Publisher
}

type publishersMap struct {
	m *sync.Map
}

func (p *publishersMap) size() (n int) {
	p.m.Range(func(key, value interface{}) bool {
		n++
		return true
	})
	return
}

func (p *publishersMap) put(id uint32, value *publishers) {
	p.m.Store(id, value)
}

func (p *publishersMap) load(id uint32) (v *publishers, ok bool) {
	found, ok := p.m.Load(id)
	if ok {
		v = found.(*publishers)
	}
	return
}

func (p *publishersMap) remove(id uint32) {
	p.m.Delete(id)
}

func newMessageStore() *publishersMap {
	return &publishersMap{
		m: &sync.Map{},
	}
}

// genStreamID is used to generate next StreamID.
type genStreamID struct {
	serverMode bool
	cur        uint32
}

func (p *genStreamID) next() uint32 {
	var v uint32
	if p.serverMode {
		// 2,4,6,8...
		v = 2 * atomic.AddUint32(&p.cur, 1)
	} else {
		// 1,3,5,7
		v = 2*(atomic.AddUint32(&p.cur, 1)-1) + 1
	}
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
