package loopywriter

import (
	"context"
	"runtime"
	"time"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/internal/queue"
	"github.com/rsocket/rsocket-go/lease"
	"github.com/rsocket/rsocket-go/logger"
)

type TransportFunc func() (*transport.Transport, error)
type WriteFunc func(*transport.Transport, core.WriteableFrame) error
type DisposeFunc func(core.WriteableFrame)

type TransportSupport interface {
	Transport() (*transport.Transport, error)
}

type LoopyWriter struct {
	q        *CtrlQueue
	tc       *core.TrafficCounter
	backlogs *queue.LKQueue
}

func NewLoopyWriter(q *CtrlQueue, enableBacklogs bool, tc *core.TrafficCounter) *LoopyWriter {
	w := &LoopyWriter{
		q:  q,
		tc: tc,
	}
	if enableBacklogs {
		w.backlogs = queue.NewLKQueue()
	}
	return w
}

func (lw *LoopyWriter) Dispose(f DisposeFunc) {
	lw.q.Dispose(f)
	if lw.backlogs == nil {
		return
	}
	for {
		next := lw.backlogs.Dequeue()
		if next == nil {
			break
		}
		wf := next.(core.WriteableFrame)
		if wf == nil || f == nil {
			continue
		}
		f(wf)
	}
}

func (lw *LoopyWriter) Run(ctx context.Context, keepalive time.Duration, ts TransportSupport) error {
	if tp, err := ts.Transport(); err != nil {
		return err
	} else if err := lw.processBacklogs(tp); err != nil {
		return err
	}
	var timeout bool
	for {
		next, le, err := lw.q.Dequeue(true, keepalive)
		timeout = err == ErrDequeueTimeout
		if err != nil && !timeout {
			return err
		}

		tp, err := ts.Transport()
		if err != nil {
			if timeout {
				return err
			}
			if lw.backlogs != nil {
				lw.backlogs.Enqueue(next)
			} else {
				next.Done()
			}
			return err
		}

		if next != nil {
			if err := lw.writeNext(tp, next); err != nil {
				if lw.backlogs != nil {
					lw.backlogs.Enqueue(next)
				} else if next != nil {
					next.Done()
				}
				return err
			}
		}
		if timeout {
			lw.writeKeepalive(tp)
		}
		if !le.IsZero() {
			lw.writeLease(tp, le)
		}
		snooze := true
	hasData:
		for {
			next, le, err = lw.q.Dequeue(false, keepalive)
			timeout = err == ErrDequeueTimeout
			if err != nil && !timeout {
				return err
			}

			if next != nil {
				if err := lw.writeNext(tp, next); err != nil {
					if lw.backlogs != nil {
						lw.backlogs.Enqueue(next)
					} else {
						next.Done()
					}
					return err
				}
				continue hasData
			}

			if timeout {
				lw.writeKeepalive(tp)
			}
			if !le.IsZero() {
				lw.writeLease(tp, le)
			}
			if snooze {
				snooze = false
				runtime.Gosched()
				continue hasData
			}
			if err := tp.Flush(); err != nil {
				return err
			}
			break hasData
		}

	}
}

func (lw *LoopyWriter) writeNext(tp *transport.Transport, next core.WriteableFrame) error {
	err := tp.Send(next, false)
	if err == nil {
		return nil
	}
	if lw.backlogs == nil {
		return err
	}
	lw.backlogs.Enqueue(next)
	return nil
}

func (lw *LoopyWriter) writeKeepalive(tp *transport.Transport) {
	next := framing.NewWriteableKeepaliveFrame(lw.tc.ReadBytes(), nil, true)
	if err := tp.Send(next, false); err != nil {
		logger.Errorf("send keepalive frame failed: %s\n", err.Error())
	}
}

func (lw *LoopyWriter) writeLease(tp *transport.Transport, l lease.Lease) {
	next := framing.NewWriteableLeaseFrame(l.TimeToLive, l.NumberOfRequests, l.Metadata)
	if err := tp.Send(next, false); err != nil {
		logger.Errorf("send lease frame failed: %s\n", err.Error())
	}
}

func (lw *LoopyWriter) processBacklogs(tp *transport.Transport) error {
	if lw.backlogs == nil {
		return nil
	}
	next := lw.backlogs.Dequeue()
	for next != nil {
		if err := tp.Send(next.(core.WriteableFrame), false); err != nil {
			return err
		}
		next = lw.backlogs.Dequeue()
	}
	return nil
}
