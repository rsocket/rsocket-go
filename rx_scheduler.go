package rsocket

var (
	schedulerElastic   Scheduler = &elasticSchedulerImpl{}
	schedulerImmediate Scheduler = &immediateSchedulerImpl{}
)

// TODO: merge codes with worker pool
type Scheduler interface {
	Do(fn func())
}

func ImmediateScheduler() Scheduler {
	return schedulerImmediate
}

func ElasticScheduler() Scheduler {
	return schedulerElastic
}

type immediateSchedulerImpl struct {
}

func (p *immediateSchedulerImpl) Do(fn func()) {
	fn()
}

type elasticSchedulerImpl struct {
}

func (p *elasticSchedulerImpl) Do(fn func()) {
	go fn()
}
