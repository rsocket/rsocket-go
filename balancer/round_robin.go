package balancer

import (
	"sync"

	"github.com/google/uuid"
	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/logger"
)

type labelClient struct {
	l string
	c rsocket.Client
}

type balancerRoundRobin struct {
	cond    *sync.Cond
	seq     int
	clients []*labelClient
	done    chan struct{}
	once    sync.Once
	onLeave []func(string)
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
	p.cond.L.Lock()
	p.clients = append(p.clients, &labelClient{
		l: label,
		c: client,
	})
	client.OnClose(func() {
		p.remove(client)
	})
	if len(p.clients) == 1 {
		p.cond.Broadcast()
	}
	p.cond.L.Unlock()
}

func (p *balancerRoundRobin) Next() (c rsocket.Client) {
	p.cond.L.Lock()
	for len(p.clients) < 1 {
		select {
		case <-p.done:
			goto L
		default:
			p.cond.Wait()
		}
	}
	c = p.choose()
L:
	p.cond.L.Unlock()
	return
}

func (p *balancerRoundRobin) choose() (cli rsocket.Client) {
	p.seq = (p.seq + 1) % len(p.clients)
	cli = p.clients[p.seq].c
	return
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
	p.cond.L.Lock()
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
	p.cond.L.Unlock()
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
		cond: sync.NewCond(&sync.Mutex{}),
		seq:  -1,
		done: make(chan struct{}),
	}
}
