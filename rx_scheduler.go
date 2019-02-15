package rsocket

import "context"

var (
	schedulerElastic   Scheduler = &elasticSchedulerImpl{}
	schedulerImmediate Scheduler = &immediateSchedulerImpl{}
)

type Do = func(ctx context.Context)

// TODO: merge codes with worker pool
type Scheduler interface {
	Do(ctx context.Context, fn Do)
}

func ImmediateScheduler() Scheduler {
	return schedulerImmediate
}

func ElasticScheduler() Scheduler {
	return schedulerElastic
}

type immediateSchedulerImpl struct {
}

func (p *immediateSchedulerImpl) Do(ctx context.Context, fn Do) {
	fn(ctx)
}

type elasticSchedulerImpl struct {
}

func (p *elasticSchedulerImpl) Do(ctx context.Context, fn Do) {
	go fn(ctx)
}
