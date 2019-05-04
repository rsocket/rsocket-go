package common

import (
	"sync"
)

// Quantile is used by load balance.
type Quantile interface {
	// Estimation returns current estimation.
	Estimation() float64
	// Insert inserts new value.
	Insert(x float64)
}

// NewFrugalQuantile returns a new quantile.
func NewFrugalQuantile(quantile, increment float64) Quantile {
	return &frugalQuantile{
		&baseFrugalQuantile{
			mutex:    &sync.Mutex{},
			incr:     increment,
			quantile: quantile,
			estimate: 0.0,
			step:     1,
			sign:     0,
		},
	}
}

// NewMedianQuantile returns a new quantile.
func NewMedianQuantile() Quantile {
	return &medianQuantile{
		&baseFrugalQuantile{
			mutex:    &sync.Mutex{},
			incr:     0.5,
			quantile: 1.0,
			estimate: 0.0,
			step:     1,
			sign:     0,
		},
	}
}

type baseFrugalQuantile struct {
	mutex    *sync.Mutex
	incr     float64
	quantile float64

	estimate   float64
	step, sign int64
}

func (p *baseFrugalQuantile) Estimation() float64 {
	return p.estimate
}

type medianQuantile struct {
	*baseFrugalQuantile
}

func (p *medianQuantile) Insert(x float64) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.sign == 0 {
		p.estimate = x
		p.sign = 1
		return
	}
	if x > p.estimate {
		p.step += p.sign

		if p.step > 0 {
			p.estimate += float64(p.step)
		} else {
			p.estimate++
		}

		if p.estimate > x {
			p.step += int64(x - p.estimate)
			p.estimate = x
		}

		if p.sign < 0 {
			p.step = 1
		}
		p.sign = 1
	} else if x < p.estimate {
		p.step -= p.sign

		if p.step > 0 {
			p.estimate -= float64(p.step)
		} else {
			p.estimate--
		}

		if p.estimate < x {
			p.step += int64(p.estimate - x)
			p.estimate = x
		}

		if p.sign > 0 {
			p.step = 1
		}
		p.sign = -1
	}
}

type frugalQuantile struct {
	*baseFrugalQuantile
}

func (p *frugalQuantile) Insert(x float64) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.sign == 0 {
		p.estimate = x
		p.sign = 1
		return
	}

	if x > p.estimate && RandFloat64() > (1-p.quantile) {
		p.step += p.sign * int64(p.incr)

		if p.step > 0 {
			p.estimate += float64(p.step)
		} else {
			p.estimate++
		}

		if p.estimate > x {
			p.step += int64(x - p.estimate)
			p.estimate = x
		}

		if p.sign < 0 {
			p.step = 1
		}
	} else if x < p.estimate && RandFloat64() > p.quantile {
		p.step -= p.sign * int64(p.incr)

		if p.step > 0 {
			p.estimate -= float64(p.step)
		} else {
			p.estimate--
		}

		if p.estimate < x {
			p.step += int64(p.estimate - x)
			p.estimate = x
		}
		if p.sign > 0 {
			p.step = 1
		}
		p.sign = -1
	}
}
