package rsocket

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/rsocket/rsocket-go/common"
	"github.com/rsocket/rsocket-go/common/logger"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

const (
	effort          = 5
	refreshDuration = 15 * time.Second
)

var (
	defaultBalancerOpts = balancerOpts{
		minActives:     3,
		maxActives:     100,
		minPendings:    1.0,
		maxPendings:    2.0,
		lowerQuantile:  0.2,
		higherQuantile: 0.8,
		expFactor:      4.0,
	}
	errAtLeastOneURI = errors.New("rsocket: at least one URI")
)

type balancerOpts struct {
	minActives, maxActives        int
	minPendings, maxPendings      float64
	lowerQuantile, higherQuantile float64
	expFactor                     float64
	initURIs                      []string
}

type balancer struct {
	locker *sync.Mutex

	discovery <-chan []string
	builder   *implClientBuilder
	actives   []*weightedSocket
	suppliers *socketSupplierPool

	targetActives                 int
	minActives, maxActives        int
	minPendings, maxPendings      float64
	expFactor                     float64
	lowerQuantile, higherQuantile common.Quantile
	pendings                      common.Ewma
	lastRefreshTime               time.Time

	done chan struct{}
	once *sync.Once
}

func (p *balancer) FireAndForget(msg payload.Payload) {
	p.next().FireAndForget(msg)
}

func (p *balancer) MetadataPush(msg payload.Payload) {
	p.next().MetadataPush(msg)
}

func (p *balancer) RequestResponse(msg payload.Payload) rx.Mono {
	return p.next().RequestResponse(msg)
}

func (p *balancer) RequestStream(msg payload.Payload) rx.Flux {
	return p.next().RequestStream(msg)
}

func (p *balancer) RequestChannel(msgs rx.Publisher) rx.Flux {
	return p.next().RequestChannel(msgs)
}

func (p *balancer) Close() (err error) {
	p.once.Do(func() {
		p.locker.Lock()
		defer p.locker.Unlock()
		var failed int
		for _, it := range p.actives {
			if err := it.Close(); err != nil {
				failed++
			}
		}
		if failed > 0 {
			err = fmt.Errorf("rsocket: close %d sockets failed", failed)
		}
		close(p.done)
	})
	return
}

func (p *balancer) watchDiscovery(ctx context.Context) error {
	var stop bool
	for {
		if stop {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-p.done:
			return nil
		case uris, ok := <-p.discovery:
			if !ok {
				stop = true
				break
			}
			if err := p.rebalance(uris...); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *balancer) rebalance(uris ...string) (err error) {
	p.locker.Lock()
	defer p.locker.Unlock()
	newest := make(map[string]*socketSupplier)
	for _, uri := range uris {
		newest[uri] = newSocketSupplier(p.builder, uri)
	}

	if len(newest) < 1 {
		err = errAtLeastOneURI
		return
	}

	logger.Infof("rsocket: rebalance with %s\n", strings.Join(uris, ","))

	removes := make([]int, 0)
	for i, l := 0, len(p.actives); i < l; i++ {
		uri := p.actives[i].supplier.u
		if _, ok := newest[uri]; !ok {
			removes = append(removes, i)
		}
	}
	for _, idx := range removes {
		rm := p.actives[idx]
		p.actives[idx] = nil
		logger.Debugf("remove then close actived socket: %s\n", rm)
		rm.lazyClose()
	}
	survives := make([]*weightedSocket, 0)
	for i, l := 0, len(p.actives); i < l; i++ {
		it := p.actives[i]
		if it != nil {
			survives = append(survives, it)
		}
	}
	p.actives = survives
	p.suppliers.reset(newest)
	return
}

func (p *balancer) next() (choose ClientSocket) {
	p.locker.Lock()
	defer p.locker.Unlock()
	p.refresh()
	n := len(p.actives)
	if n < 1 {
		choose = defaultMustFailedSocket
		return
	}
	if n == 1 {
		choose = p.actives[0]
		return
	}

	var rsc1, rsc2 *weightedSocket
	for i := 0; i < effort; i++ {
		i1 := common.RandIntn(n)
		i2 := common.RandIntn(n - 1)
		if i2 >= i1 {
			i2++
		}
		rsc1 = p.actives[i1]
		rsc2 = p.actives[i2]
		if rsc1.availability > 0 && rsc2.availability > 0 {
			break
		}
		if i+1 == effort && p.suppliers.size() > 0 {
			p.acquire()
		}
	}

	if rsc1 == nil && rsc2 == nil {
		if len(p.actives) > 0 {
			choose = p.actives[0]
		} else {
			choose = defaultMustFailedSocket
		}
		return
	}
	w1 := p.algorithmicWeight(rsc1)
	w2 := p.algorithmicWeight(rsc2)
	if w1 < w2 {
		logger.Debugf("choose=%s, giveup=%s, %.8f > %.8f\n", rsc2, rsc1, w2, w1)
		choose = rsc2
	} else {
		logger.Debugf("choose=%s, giveup=%s, %.8f > %.8f\n", rsc1, rsc2, w1, w2)
		choose = rsc1
	}
	return
}

func (p *balancer) refreshTargetActives() {
	n := len(p.actives)
	if n == 0 {
		return
	}
	var sum float64
	for _, it := range p.actives {
		sum += float64(it.pending)
	}
	p.pendings.Insert(sum / float64(n))
	now := time.Now()
	if now.Sub(p.lastRefreshTime) <= refreshDuration {
		return
	}
	avg := p.pendings.Value()
	if avg < 1.0 {
		p.targetActives--
	} else if avg > 2.0 {
		p.targetActives++
	} else {
		return
	}

	if p.targetActives < p.minActives {
		p.targetActives = p.minActives
	}
	realMax := p.maxActives
	if v := n + p.suppliers.size(); v < realMax {
		realMax = v
	}
	if p.targetActives > realMax {
		p.targetActives = realMax
	}
	p.lastRefreshTime = now
	p.pendings.Reset((p.minPendings + p.maxPendings) / 2)
}

func (p *balancer) refresh() {
	p.refreshTargetActives()
	// no more free supplier.
	if p.suppliers.size() < 1 {
		return
	}
	n := len(p.actives)
	switch {
	case n > p.targetActives:
		// decr a socket
		p.release()
	case n < p.targetActives:
		// incr a socket
		p.acquire()
	}
}

func (p *balancer) release() {
	minWeight := math.MaxFloat64
	removed := -1
	n := len(p.actives)
	if n < 2 {
		return
	}
	for i := 0; i < n; i++ {
		v := p.actives[i]
		w := p.algorithmicWeight(v)
		if w < minWeight {
			minWeight = w
			removed = i
		}
	}
	if removed < 0 {
		return
	}
	dead := p.actives[removed]
	p.actives[removed] = p.actives[n-1]
	p.actives = p.actives[:n-1]

	// return supplier
	p.suppliers.returnSupplier(dead.supplier)
	dead.lazyClose()
}

func (p *balancer) acquire() bool {
	supplier, ok := p.suppliers.next()
	if !ok {
		logger.Debugf("rsocket: no socket supplier available\n")
		return false
	}
	logger.Debugf("choose supplier %s\n", supplier)
	sk, err := supplier.create(p.lowerQuantile, p.higherQuantile)
	if err != nil {
		_ = p.suppliers.returnSupplier(supplier)
		return false
	}
	p.actives = append(p.actives, sk)

	// TODO: ugly code
	merge := &struct {
		sk *weightedSocket
		ba *balancer
	}{sk, p}
	sk.origin.(*duplexRSocket).tp.OnClose(func() {
		merge.ba.locker.Lock()
		defer merge.ba.locker.Unlock()
		logger.Debugf("rsocket: unload %s\n", merge.sk)
		if ok := merge.ba.unload(merge.sk); ok {
			_ = merge.ba.suppliers.returnSupplier(merge.sk.supplier)
		}
	})
	return true
}

func (p *balancer) unload(socket *weightedSocket) bool {
	idx := -1
	for i := 0; i < len(p.actives); i++ {
		if p.actives[i] == socket {
			idx = i
			break
		}
	}
	if idx < 0 {
		return false
	}
	p.actives[idx] = p.actives[len(p.actives)-1]
	p.actives[len(p.actives)-1] = nil
	p.actives = p.actives[:len(p.actives)-1]
	return true
}

func (p *balancer) algorithmicWeight(socket *weightedSocket) float64 {
	if socket == nil || socket.availability == 0 {
		return 0
	}
	pendings := float64(socket.pending)
	latency := socket.getPredictedLatency()
	//logger.Infof("%s: latency=%f\n", socket, latency)
	low := p.lowerQuantile.Estimation()
	high := math.Max(p.higherQuantile.Estimation(), low*1.001)
	bandWidth := math.Max(high-low, 1)
	if latency < low {
		alpha := (low - latency) / bandWidth
		bonusFactor := math.Pow(1+alpha, p.expFactor)
		latency /= bonusFactor
	} else if latency > high {
		alpha := (latency - high) / bandWidth
		penaltyFactor := math.Pow(1+alpha, p.expFactor)
		latency *= penaltyFactor
	}
	w := socket.availability / (1 + latency*(pendings+1))
	/*logger.Infof("%s: w=%f, high=%f, low=%f, availability=%f, pending=%f, latency=%f, \n",
	socket, w,
	high, low,
	socket.availability, pendings, latency)*/
	return w
}

func newBalancer(bu *implClientBuilder, discovery <-chan []string, o *balancerOpts) *balancer {
	var pool *socketSupplierPool
	if len(o.initURIs) > 0 {
		suppliers := make([]*socketSupplier, 0)
		for _, value := range o.initURIs {
			suppliers = append(suppliers, newSocketSupplier(bu, value))
		}
		pool = newSocketPool(suppliers...)
	} else {
		pool = newSocketPool()
	}
	return &balancer{
		discovery:       discovery,
		builder:         bu,
		locker:          &sync.Mutex{},
		actives:         make([]*weightedSocket, 0),
		suppliers:       pool,
		done:            make(chan struct{}, 0),
		targetActives:   o.minActives,
		minActives:      o.minActives,
		maxActives:      o.maxActives,
		lowerQuantile:   common.NewFrugalQuantile(o.lowerQuantile, 1),
		higherQuantile:  common.NewFrugalQuantile(o.higherQuantile, 1),
		expFactor:       o.expFactor,
		once:            &sync.Once{},
		lastRefreshTime: time.Now(),
		pendings:        common.NewEwma(15, time.Second, (o.minPendings+o.maxPendings)/2.0),
	}
}

func newBalancerStarter(bu *implClientBuilder, discovery <-chan []string, options ...OptBalancer) *balancerStarter {
	return &balancerStarter{
		bu:        bu,
		opts:      options,
		discovery: discovery,
	}
}

type balancerStarter struct {
	bu        *implClientBuilder
	opts      []OptBalancer
	discovery <-chan []string
}

func (p *balancerStarter) Start() (ClientSocket, error) {
	opts := defaultBalancerOpts
	for _, fn := range p.opts {
		fn(&opts)
	}
	ba := newBalancer(p.bu, p.discovery, &opts)
	go func(ctx context.Context) {
		if err := ba.watchDiscovery(ctx); err != nil {
			logger.Warnf("reblance exit with error: %s\n", err.Error())
		}
	}(context.Background())
	return ba, nil
}

// OptBalancer can be used to set options for balancer.
type OptBalancer func(opts *balancerOpts)

// WithInitTransports sets initial transport URI.
func WithInitTransports(uris ...string) OptBalancer {
	return func(opts *balancerOpts) {
		opts.initURIs = uris
	}
}

// WithQuantile sets quantile range of a balancer. (default: 0.2 ~ 0.8)
func WithQuantile(lower, higher float64) OptBalancer {
	return func(opts *balancerOpts) {
		opts.lowerQuantile = lower
		opts.higherQuantile = higher
	}
}

// WithPendings sets pendings range for a balancer. (default: 1.0 ~ 2.0)
func WithPendings(min, max float64) OptBalancer {
	return func(opts *balancerOpts) {
		opts.minPendings = min
		opts.maxPendings = max
	}
}

// WithActives limit amount of active sockets for a balancer. (default: 3 ~ 100)
func WithActives(min, max int) OptBalancer {
	return func(opts *balancerOpts) {
		opts.minActives = min
		opts.maxActives = max
	}
}
