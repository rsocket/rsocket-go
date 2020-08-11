package balancer

import (
	"errors"
	"sync"

	"github.com/rsocket/rsocket-go/logger"
)

var errGroupClosed = errors.New("balancer group has been closed")

// Group manage a group of Balancer.
// Group can be used to create a simple RSocket Broker.
type Group struct {
	g func() Balancer
	l sync.Mutex
	m map[string]Balancer
}

// Close close current RSocket group.
func (p *Group) Close() (err error) {
	if p.m == nil {
		return
	}
	all := make(chan Balancer)
	done := make(chan struct{})
	go func(all chan Balancer, done chan struct{}) {
		defer func() {
			close(done)
		}()
		for it := range all {
			if err := it.Close(); err != nil {
				logger.Warnf("close balancer failed: %s\n", err)
			}
		}
	}(all, done)

	p.l.Lock()
	defer p.l.Unlock()
	for _, b := range p.m {
		all <- b
	}
	p.m = nil
	close(all)
	<-done
	return
}

// Get returns a Balancer with custom id.
func (p *Group) Get(id string) Balancer {
	p.l.Lock()
	defer p.l.Unlock()
	if p.m == nil {
		panic(errGroupClosed)
	}
	if actual, ok := p.m[id]; ok {
		return actual
	}
	newborn := p.g()
	p.m[id] = newborn
	return newborn
}

// NewGroup returns a new Group.
func NewGroup(gen func() Balancer) *Group {
	return &Group{
		g: gen,
		m: make(map[string]Balancer),
	}
}
