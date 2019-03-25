package rx

import (
	"context"
	"github.com/panjf2000/ants"
	"io"
)

var (
	elasticScheduler   = NewElasticScheduler(ants.DefaultAntsPoolSize)
	immediateScheduler = &immediateSchedulerImpl{}
)

// Do is alias of the function which will be executed in scheduler.
type Do = func(ctx context.Context)

// Scheduler is a work pool for do soming async.
type Scheduler interface {
	io.Closer
	// Do register function to do.
	Do(ctx context.Context, fn Do)
}

// ImmediateScheduler returns a scheduler which will be executed immediate.
func ImmediateScheduler() Scheduler {
	return immediateScheduler
}

// ElasticScheduler returns a dynamic scheduler.
func ElasticScheduler() Scheduler {
	return elasticScheduler
}

// NewElasticScheduler returns a new ElasticScheduler.
func NewElasticScheduler(size int) Scheduler {
	pool, err := ants.NewPool(size)
	if err != nil {
		panic(err)
	}
	return &elasticSchedulerImpl{
		pool: pool,
	}
}

type immediateSchedulerImpl struct {
}

func (p *immediateSchedulerImpl) Close() error {
	return nil
}

func (p *immediateSchedulerImpl) Do(ctx context.Context, fn Do) {
	fn(ctx)
}

type elasticSchedulerImpl struct {
	pool *ants.Pool
}

func (p *elasticSchedulerImpl) Close() error {
	return p.pool.Release()
}

func (p *elasticSchedulerImpl) Do(ctx context.Context, fn Do) {
	err := p.pool.Submit(func() {
		fn(ctx)
	})
	if err != nil {
		panic(err)
	}
}
