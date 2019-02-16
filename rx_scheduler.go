package rsocket

import (
	"context"
	"github.com/panjf2000/ants"
	_ "github.com/panjf2000/ants"
	"io"
)

var (
	elasticScheduler   = NewElasticScheduler(ants.DefaultAntsPoolSize)
	immediateScheduler = &immediateSchedulerImpl{}
)

type Do = func(ctx context.Context)

type Scheduler interface {
	io.Closer
	Do(ctx context.Context, fn Do)
}

func ImmediateScheduler() Scheduler {
	return immediateScheduler
}

func ElasticScheduler() Scheduler {
	return elasticScheduler
}

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
