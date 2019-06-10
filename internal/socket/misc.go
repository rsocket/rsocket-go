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

type mailboxMode int8

const (
	mailRequestResponse mailboxMode = iota
	mailRequestStream
	mailRequestChannel
)

var mailboxPool = sync.Pool{
	New: func() interface{} {
		return new(mailbox)
	},
}

// SetupInfo represents basic info of setup.
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

// ToFrame converts current SetupInfo to a frame of Setup.
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

func borrowPublishers(mode mailboxMode, sending, receiving rx.Publisher) (b *mailbox) {
	b = mailboxPool.Get().(*mailbox)
	b.mode = mode
	b.sending = sending
	b.receiving = receiving
	return
}

func returnPublishers(b *mailbox) {
	b.receiving = nil
	b.sending = nil
	mailboxPool.Put(b)
}

type mailbox struct {
	mode               mailboxMode
	sending, receiving rx.Publisher
}

type mailboxes struct {
	locker *sync.RWMutex
	m      map[uint32]*mailbox
}

func (p *mailboxes) each(fn func(id uint32, elem *mailbox)) {
	p.locker.RLock()
	defer p.locker.RUnlock()
	for k, v := range p.m {
		fn(k, v)
	}
}

func (p *mailboxes) put(id uint32, mode mailboxMode, sending, receiving rx.Publisher) {
	p.locker.Lock()
	p.m[id] = borrowPublishers(mode, sending, receiving)
	p.locker.Unlock()
}

func (p *mailboxes) load(id uint32) (v *mailbox, ok bool) {
	p.locker.RLock()
	v, ok = p.m[id]
	p.locker.RUnlock()
	return
}

func (p *mailboxes) remove(id uint32) {
	p.locker.Lock()
	found, ok := p.m[id]
	if ok {
		delete(p.m, id)
		p.locker.Unlock()
		returnPublishers(found)
	} else {
		p.locker.Unlock()
	}
}

func newMailboxes() *mailboxes {
	return &mailboxes{
		locker: &sync.RWMutex{},
		m:      make(map[uint32]*mailbox),
	}
}

type streamIDs interface {
	next() uint32
}

type serverStreamIDs struct {
	cur uint32
}

func (p *serverStreamIDs) next() uint32 {
	// 2,4,6,8...
	v := 2 * atomic.AddUint32(&p.cur, 1)
	if v != 0 {
		return v
	}
	return p.next()
}

type clientStreamIDs struct {
	cur uint32
}

func (p *clientStreamIDs) next() uint32 {
	// 1,3,5,7
	v := 2*(atomic.AddUint32(&p.cur, 1)-1) + 1
	if v != 0 {
		return v
	}
	return p.next()
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
