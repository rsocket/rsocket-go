package common

import (
	"sync"
)

// Quantile is used by load balance.
type Quantile interface {
	// Estimation returns current estimation.
	Estimation() float32
	// Insert inserts new value.
	Insert(x float32)
}

// NewFrugalQuantile returns a new quantile.
func NewFrugalQuantile(quantile, increment float32) Quantile {
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
	incr     float32
	quantile float32

	estimate   float32
	step, sign int32
}

func (p *baseFrugalQuantile) Estimation() float32 {
	return p.estimate
}

type medianQuantile struct {
	*baseFrugalQuantile
}

func (p *medianQuantile) Insert(x float32) {
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
			p.estimate += float32(p.step)
		} else {
			p.estimate++
		}

		if p.estimate > x {
			p.step += int32(x - p.estimate)
			p.estimate = x
		}

		if p.sign < 0 {
			p.step = 1
		}
		p.sign = 1
	} else if x < p.estimate {
		p.step -= p.sign

		if p.step > 0 {
			p.estimate -= float32(p.step)
		} else {
			p.estimate--
		}

		if p.estimate < x {
			p.step += int32(p.estimate - x)
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

func (p *frugalQuantile) Insert(x float32) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if p.sign == 0 {
		p.estimate = x
		p.sign = 1
		return
	}

	if x > p.estimate && defaultRand.Float32() > (1-p.quantile) {
		p.step += p.sign * int32(p.incr)

		if p.step > 0 {
			p.estimate += float32(p.step)
		} else {
			p.estimate++
		}

		if p.estimate > x {
			p.step += int32(x - p.estimate)
			p.estimate = x
		}

		if p.sign < 0 {
			p.step = 1
		}
	} else if x < p.estimate && defaultRand.Float32() > p.quantile {
		p.step -= p.sign * int32(p.incr)

		if p.step > 0 {
			p.estimate -= float32(p.step)
		} else {
			p.estimate--
		}

		if p.estimate < x {
			p.step += int32(p.estimate - x)
			p.estimate = x
		}
		if p.sign > 0 {
			p.step = 1
		}
		p.sign = -1
	}
}
