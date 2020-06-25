package balancer

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/logger"
	"go.uber.org/atomic"
)

type labelClient struct {
	l string
	c rsocket.Client
}

type balancerRoundRobin struct {
	seq     *atomic.Uint32
	mutex   sync.RWMutex
	clients []*labelClient
	done    chan struct{}
	once    sync.Once
	onLeave []func(string)
	cond    *sync.Cond
}

func (p *balancerRoundRobin) OnLeave(fn func(label string)) {
	if fn != nil {
		p.onLeave = append(p.onLeave, fn)
	}
}

func (p *balancerRoundRobin) Put(client rsocket.Client) {
	label := uuid.New().String()
	p.PutLabel(label, client)
}

func (p *balancerRoundRobin) PutLabel(label string, client rsocket.Client) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.clients = append(p.clients, &labelClient{
		l: label,
		c: client,
	})
	client.OnClose(func(error) {
		p.remove(client)
	})
	if len(p.clients) == 1 {
		p.cond.Broadcast()
	}
}

func (p *balancerRoundRobin) MustNext(ctx context.Context) rsocket.Client {
	c, ok := p.Next(ctx)
	if !ok {
		panic("cannot get next client from current balancer")
	}
	return c
}

func (p *balancerRoundRobin) Next(ctx context.Context) (rsocket.Client, bool) {
	p.mutex.RLock()
	defer p.mutex.RUnlock()
	if n := len(p.clients); n > 0 {
		idx := int(p.seq.Inc() % uint32(n))
		return p.clients[idx].c, true
	}

	ch := make(chan rsocket.Client, 1)
	closed := atomic.NewBool(false)

	go func() {
		p.cond.L.Lock()
		for len(p.clients) < 1 && !closed.Load() {
			p.cond.Wait()
		}
		p.cond.L.Unlock()
		if n := len(p.clients); n > 0 {
			idx := int(p.seq.Inc() % uint32(n))
			ch <- p.clients[idx].c
		}
	}()

	select {
	case <-ctx.Done():
		closed.Store(true)
		p.cond.Broadcast()
		return nil, false
	case c, ok := <-ch:
		if !ok {
			return nil, false
		}
		return c, true
	}
}

func (p *balancerRoundRobin) Close() (err error) {
	p.once.Do(func() {
		if len(p.clients) < 1 {
			return
		}
		clone := append([]*labelClient(nil), p.clients...)
		close(p.done)
		wg := &sync.WaitGroup{}
		wg.Add(len(clone))
		for _, value := range clone {
			go func(c rsocket.Client, wg *sync.WaitGroup) {
				defer wg.Done()
				if err := c.Close(); err != nil {
					logger.Warnf("close client failed: %s\n", err)
				}
			}(value.c, wg)
		}
		wg.Wait()
	})
	return
}

func (p *balancerRoundRobin) remove(client rsocket.Client) (label string, ok bool) {
	p.mutex.Lock()
	j := -1
	for i, l := 0, len(p.clients); i < l; i++ {
		if p.clients[i].c == client {
			j = i
			break
		}
	}
	ok = j > -1
	if ok {
		label = p.clients[j].l
		p.clients = append(p.clients[:j], p.clients[j+1:]...)
	}
	p.mutex.Unlock()
	if ok && len(p.onLeave) > 0 {
		go func(label string) {
			for _, fn := range p.onLeave {
				fn(label)
			}
		}(label)
	}
	return
}

// NewRoundRobinBalancer returns a new Round-Robin Balancer.
func NewRoundRobinBalancer() Balancer {
	return &balancerRoundRobin{
		cond: sync.NewCond(new(sync.Mutex)),
		seq:  atomic.NewUint32(0),
		done: make(chan struct{}),
	}
}
