package complex

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/rsocket/rsocket-go"
)

const epsilon = 1e-4

type socketSupplier struct {
	mutex *sync.Mutex

	u string
	b rsocket.ClientBuilder

	tau      int64
	stamp    int64
	accuracy Ewma
}

func (p *socketSupplier) String() string {
	return fmt.Sprintf("SocketSupplier{transport=%s, v=%.2f}", p.u, p.availability())
}

func (p *socketSupplier) create(lower, higher Quantile) (socket *weightedSocket, err error) {
	var v float64
	var origin rsocket.Client
	origin, err = rsocket.Start(context.Background())
	if err == nil {
		socket = newWeightedSocket(lower, higher, origin, p)
		v = 1
	}
	p.updateAccuracy(v)
	return
}

func (p *socketSupplier) availability() float64 {
	e := Value()
	if NowInMicrosecond()-p.stamp > p.tau {
		a := math.Min(1.0, e+0.5)
		p.mutex.Lock()
		Reset(a)
		p.mutex.Unlock()
	}
	if e < epsilon {
		e = 0
	} else if 1-epsilon < e {
		e = 1
	}
	return e
}

func (p *socketSupplier) updateAccuracy(v float64) {
	p.mutex.Lock()
	Insert(v)
	p.stamp = NowInMicrosecond()
	p.mutex.Unlock()
}

func newSocketSupplier(builder rsocket.ClientBuilder, uri string) *socketSupplier {
	return &socketSupplier{
		mutex:    &sync.Mutex{},
		u:        uri,
		b:        builder,
		tau:      CalcTAU(5, time.Second),
		stamp:    NowInMicrosecond(),
		accuracy: NewEwma(5, time.Second, 1.0),
	}
}
