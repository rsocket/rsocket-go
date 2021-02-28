package loopywriter

import (
	"context"
	"io"
	"runtime"
	"time"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/core/transport"
	"github.com/rsocket/rsocket-go/internal/queue"
	"github.com/rsocket/rsocket-go/internal/tpfactory"
	"github.com/rsocket/rsocket-go/lease"
	"github.com/rsocket/rsocket-go/logger"
)

type TransportFunc func() (*transport.Transport, error)
type WriteFunc func(*transport.Transport, core.WriteableFrame) error
type DisposeFunc func(core.WriteableFrame)

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

func (lw *LoopyWriter) Run(ctx context.Context, keepalive time.Duration, factory *tpfactory.TransportFactory) error {
	// Drain all of backlog frames
	if err := lw.drainBacklogs(ctx, factory); err != nil {
		return err
	}

	var (
		timeout   bool
		tp        *transport.Transport
		next      core.WriteableFrame
		nextLease lease.Lease
		err       error
	)

	for {
		// dequeue next data
		next, nextLease, err = lw.q.Dequeue(true, keepalive)
		timeout = err == ErrDequeueTimeout
		if err != nil && !timeout {
			if err == io.EOF {
				return nil
			}
			return err
		}

		// try get available transport
		tp, err = factory.Get(ctx, false)
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

		// write the next frame
		if next != nil {
			err = lw.writeNext(tp, next)
			if err != nil {
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
		if !nextLease.IsZero() {
			lw.writeLease(tp, nextLease)
		}

		// init snooze with true
		snooze := true
	hasData:
		for {
			next, nextLease, err = lw.q.Dequeue(false, keepalive)
			timeout = err == ErrDequeueTimeout
			if err != nil && !timeout {
				return err
			}

			if next != nil {
				err = lw.writeNext(tp, next)
				if err != nil {
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
			if !nextLease.IsZero() {
				lw.writeLease(tp, nextLease)
			}
			if snooze {
				snooze = false
				runtime.Gosched()
				continue hasData
			}
			err = tp.Flush()
			if err != nil {
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

func (lw *LoopyWriter) drainBacklogs(ctx context.Context, tf *tpfactory.TransportFactory) error {
	tp, err := tf.Get(ctx, true)
	if err != nil {
		return err
	}
	if lw.backlogs == nil {
		return nil
	}
	next := lw.backlogs.Dequeue()
	for next != nil {
		if e := tp.Send(next.(core.WriteableFrame), false); e != nil {
			return e
		}
		next = lw.backlogs.Dequeue()
	}
	return nil
}
