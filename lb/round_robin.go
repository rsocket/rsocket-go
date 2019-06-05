package lb

import (
	"sync"

	"github.com/rsocket/rsocket-go"
	"github.com/rsocket/rsocket-go/internal/logger"
)

type balancerRoundRobin struct {
	cond    *sync.Cond
	seq     int
	clients []rsocket.Client
	done    chan struct{}
	once    *sync.Once
}

func (p *balancerRoundRobin) Push(client rsocket.Client) {
	p.cond.L.Lock()
	p.clients = append(p.clients, client)
	if len(p.clients) == 1 {
		p.cond.Broadcast()
	}
	p.cond.L.Unlock()
}

func (p *balancerRoundRobin) Next() (client rsocket.Client, ok bool) {
	p.cond.L.Lock()
	for len(p.clients) < 1 {
		select {
		case <-p.done:
			goto L
		default:
			p.cond.Wait()
		}
	}
	client = p.choose()
L:
	p.cond.L.Unlock()
	return
}

func (p *balancerRoundRobin) choose() rsocket.Client {
	p.seq = (p.seq + 1) % len(p.clients)
	return p.clients[p.seq]
}

func (p *balancerRoundRobin) Close() (err error) {
	p.once.Do(func() {
		if len(p.clients) < 1 {
			return
		}
		clone := append([]rsocket.Client(nil), p.clients...)
		close(p.done)
		wg := &sync.WaitGroup{}
		wg.Add(len(clone))
		for _, value := range clone {
			go func(c rsocket.Client, wg *sync.WaitGroup) {
				defer wg.Done()
				if err := c.Close(); err != nil {
					logger.Warnf("close client failed: %s\n", err)
				}
			}(value, wg)
		}
		wg.Wait()
	})
	return
}

func NewRoundRobin() Balancer {
	return &balancerRoundRobin{
		cond: sync.NewCond(&sync.Mutex{}),
		seq:  -1,
		once: &sync.Once{},
		done: make(chan struct{}),
	}
}
