package socket

import (
	"time"

	"github.com/rsocket/rsocket-go/lease"
	"go.uber.org/atomic"
)

type leaser struct {
	deadline    *atomic.Int64
	tickets     *atomic.Int64
	initialized *atomic.Bool
}

func (p *leaser) refresh(deadline time.Time, tickets int64) {
	if p != nil {
		p.deadline.Store(deadline.UnixNano())
		p.tickets.Store(tickets)
		p.initialized.Store(true)
	}
}

func (p *leaser) allow() (err error) {
	if p == nil {
		return
	}
	if !p.initialized.Load() {
		err = lease.ErrLeaseNotRcv
	} else if time.Now().UnixNano() > p.deadline.Load() {
		err = lease.ErrLeaseExpired
	} else if p.tickets.Dec() < 0 {
		err = lease.ErrLeaseNoMoreRequests
	}
	return
}

//func (p *leaser) allowFrame(f framing.Frame) (err error) {
//	if p == nil {
//		return
//	}
//	header := f.Header()
//	if header.Flag().Check(framing.FlagFollow) {
//		return
//	}
//	switch header.Type() {
//	case framing.FrameTypeRequestFNF, framing.FrameTypeRequestResponse, framing.FrameTypeRequestStream, framing.FrameTypeRequestChannel:
//		err = p.allow()
//	}
//	return
//}

func newLeaser(deadline time.Time, n int64) *leaser {
	return &leaser{
		deadline:    atomic.NewInt64(deadline.UnixNano()),
		tickets:     atomic.NewInt64(n),
		initialized: atomic.NewBool(false),
	}
}
