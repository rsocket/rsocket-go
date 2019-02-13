package rsocket

import (
	"io"
	"sync"
)

var infinite = &infiniteWorkerPool{}

type workerPool interface {
	io.Closer
	Do(fn func())
}

type infiniteWorkerPool struct {
}

func (p *infiniteWorkerPool) Close() error {
	return nil
}

func (p *infiniteWorkerPool) Do(fn func()) {
	go fn()
}

type fixedWorkerPool struct {
	n            int
	jobs         chan func()
	onceRun      *sync.Once
	onceClose    *sync.Once
	waitingClose *sync.WaitGroup
}

func (p *fixedWorkerPool) Close() error {
	p.onceClose.Do(func() {
		close(p.jobs)
		p.waitingClose.Wait()
	})
	return nil
}

func (p *fixedWorkerPool) Do(fn func()) {
	p.jobs <- fn
}

func (p *fixedWorkerPool) run() {
	p.onceRun.Do(func() {
		for i := 0; i < p.n; i++ {
			go p.loop()
		}
	})
}

func (p *fixedWorkerPool) loop() {
	defer p.waitingClose.Done()
	p.waitingClose.Add(1)
	var stop bool
	for {
		if stop {
			break
		}
		select {
		case fn, ok := <-p.jobs:
			if !ok {
				stop = true
				break
			}
			fn()
		}
	}
}

func newWorkerPool(n int) workerPool {
	if n < 1 {
		return infinite
	}
	pool := &fixedWorkerPool{
		n:            n,
		jobs:         make(chan func(), n*2),
		onceClose:    &sync.Once{},
		onceRun:      &sync.Once{},
		waitingClose: &sync.WaitGroup{},
	}
	defer pool.run()
	return pool
}
