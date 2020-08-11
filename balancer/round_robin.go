package balancer

import (
	"context"
	"runtime"
	"sync"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/logger"
	"go.uber.org/atomic"
)

var errConflictSocket = errors.New("socket exists already")

type balancerRoundRobin struct {
	seq     *atomic.Uint32
	keys    []string
	sockets []rsocket.Client
	done    chan struct{}
	once    sync.Once
	onLeave []func(string)
	c       *common.Cond
}

func (b *balancerRoundRobin) OnLeave(fn func(label string)) {
	if fn != nil {
		b.onLeave = append(b.onLeave, fn)
	}
}

func (b *balancerRoundRobin) Put(client rsocket.Client) error {
	return b.PutLabel(uuid.New().String(), client)
}

func (b *balancerRoundRobin) PutLabel(label string, client rsocket.Client) error {
	b.c.L.Lock()
	defer b.c.L.Unlock()
	for _, k := range b.keys {
		if k == label {
			return errConflictSocket
		}
	}
	b.keys = append(b.keys, label)
	b.sockets = append(b.sockets, client)
	if n := len(b.sockets); n == 1 {
		b.c.Broadcast()
	}
	client.OnClose(func(err error) {
		b.remove(client)
	})
	return nil
}

func (b *balancerRoundRobin) Next(ctx context.Context) (client rsocket.Client, ok bool) {
	b.c.L.Lock()
	for {
		n := len(b.keys)
		if n > 0 {
			idx := int(b.seq.Inc() % uint32(n))
			client = b.sockets[idx]
			ok = true
			break
		}
		if b.c.Wait(ctx) {
			break
		}
		b.c.L.Unlock()
		runtime.Gosched()
		b.c.L.Lock()
	}
	b.c.L.Unlock()
	return
}

func (b *balancerRoundRobin) Close() (err error) {
	b.once.Do(func() {
		if len(b.sockets) < 1 {
			return
		}
		clone := append([]rsocket.Client(nil), b.sockets...)
		close(b.done)
		wg := &sync.WaitGroup{}
		wg.Add(len(clone))
		for i := 0; i < len(clone); i++ {
			go func(c rsocket.Client, wg *sync.WaitGroup) {
				defer wg.Done()
				if err := c.Close(); err != nil {
					logger.Warnf("close client failed: %s\n", err)
				}
			}(clone[i], wg)
		}
		wg.Wait()
	})
	return
}

func (b *balancerRoundRobin) remove(client rsocket.Client) (label string, ok bool) {
	b.c.L.Lock()
	j := -1
	for i, l := 0, len(b.sockets); i < l; i++ {
		if b.sockets[i] == client {
			j = i
			break
		}
	}
	ok = j > -1
	if ok {
		label = b.keys[j]
		b.keys = append(b.keys[:j], b.keys[j+1:]...)
		b.sockets = append(b.sockets[:j], b.sockets[j+1:]...)
	}
	b.c.L.Unlock()
	if ok && len(b.onLeave) > 0 {
		go func(label string) {
			for _, fn := range b.onLeave {
				fn(label)
			}
		}(label)
	}
	return
}

// NewRoundRobinBalancer returns a new Round-Robin Balancer.
func NewRoundRobinBalancer() Balancer {
	return &balancerRoundRobin{
		seq:  atomic.NewUint32(0),
		done: make(chan struct{}),
		c:    common.NewCond(&sync.Mutex{}),
	}
}
