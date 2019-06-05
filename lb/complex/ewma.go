package complex

import (
	"fmt"
	"math"
	"time"
)

var halfLifeDiv = math.Log(2)

// Ewma is used to compute the exponential weighted moving average of a series of values
type Ewma interface {
	// Insert inserts a value.
	Insert(x float64)
	// Reset reset current value.
	Reset(x float64)
	// Value returns current value.
	Value() float64
}

// NowInMicrosecond returns UNIX time in microsecond.
func NowInMicrosecond() int64 {
	return time.Now().UnixNano() / 1000
}

// CalcTAU returns tau.
func CalcTAU(halfLife float64, unit time.Duration) int64 {
	tau := unit / (time.Duration(halfLife/halfLifeDiv) * time.Microsecond)
	return int64(tau)
}

// NewEwma returns a new ewma.
func NewEwma(halfLife float64, unit time.Duration, initialValue float64) Ewma {
	return &implEwma{
		tau:   float64(CalcTAU(halfLife, unit)),
		stamp: 0,
		ewma:  initialValue,
	}
}

type implEwma struct {
	tau   float64
	stamp int64
	ewma  float64
}

func (p *implEwma) String() string {
	return fmt.Sprintf("Ewma{value=%.4f,age=%d}", p.ewma, time.Now().UnixNano()/1000-p.stamp)
}

func (p *implEwma) Insert(x float64) {
	now := NowInMicrosecond()
	var elapsed float64
	if d := now - p.stamp; d > 0 {
		elapsed = float64(d)
	}
	p.stamp = now
	w := math.Exp(-elapsed / p.tau)
	p.ewma = w*p.ewma + (1-w)*x
}

func (p *implEwma) Reset(x float64) {
	p.stamp = 0
	p.ewma = x
}

func (p *implEwma) Value() float64 {
	return p.ewma
}
