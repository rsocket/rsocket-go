package rsocket

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/rsocket/rsocket-go/common/logger"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

const (
	defaultMinActives = 3
	defaultMaxActives = 100
)

var (
	errMustFailed          = errors.New("must failed")
	errAtLeastTwoSuppliers = errors.New("rsocket: at least two clients in balancer")
	mustFailed             = &mustFailedSocket{}
)

type balancer struct {
	seq                    int
	mu                     *sync.Mutex
	actives                []*availabilitySocket
	suppliers              *socketSupplierPool
	minActives, maxActives int
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
	var failed int
	for _, it := range p.actives {
		if err := it.Close(); err != nil {
			failed++
		}
	}
	if failed > 0 {
		err = fmt.Errorf("rsocket: close %d sockets failed", failed)
	}
	return
}

func (p *balancer) next() ClientSocket {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.refresh()
	n := len(p.actives)
	if n < 1 {
		return mustFailed
	}
	// TODO: support more lb algorithm.
	// simple RoundRobin
	p.seq = (p.seq + 1) % n
	return p.actives[p.seq]
}

func (p *balancer) refresh() {
	n := len(p.actives)
	switch {
	case n > p.maxActives:
		// TODO: reduce active sockets
	case n < p.minActives:
		d := p.minActives - n
		for i := 0; i < d && p.suppliers.Len() > 0; i++ {
			p.acquire()
		}
	}
}

func (p *balancer) acquire() bool {
	supplier, ok := p.suppliers.Next()
	if !ok {
		logger.Debugf("rsocket: no socket supplier available\n")
		return false
	}
	sk, err := supplier.create()
	if err != nil {
		_ = p.suppliers.returnSupplier(supplier)
		return false
	}
	p.actives = append(p.actives, sk)
	// TODO: ugly code
	merge := &struct {
		sk *availabilitySocket
		ba *balancer
	}{sk, p}
	sk.origin.(*duplexRSocket).tp.OnClose(func() {
		merge.ba.mu.Lock()
		defer merge.ba.mu.Unlock()
		logger.Infof("rsocket: unload %s\n", merge.sk)
		merge.ba.unload(merge.sk)
		_ = merge.ba.suppliers.returnSupplier(merge.sk.supplier)
	})
	return true
}

func (p *balancer) unload(socket *availabilitySocket) {
	idx := -1
	for i := 0; i < len(p.actives); i++ {
		if p.actives[i] == socket {
			idx = i
			break
		}
	}
	if idx < 0 {
		return
	}
	p.actives[idx] = p.actives[len(p.actives)-1]
	p.actives[len(p.actives)-1] = nil
	p.actives = p.actives[:len(p.actives)-1]
}

type mustFailedSocket struct {
}

func (p *mustFailedSocket) Close() error {
	return nil
}

func (*mustFailedSocket) FireAndForget(msg payload.Payload) {
	msg.Release()
}

func (*mustFailedSocket) MetadataPush(msg payload.Payload) {
	msg.Release()
}

func (*mustFailedSocket) RequestResponse(msg payload.Payload) rx.Mono {
	return rx.
		NewMono(func(ctx context.Context, sink rx.MonoProducer) {
			sink.Error(errMustFailed)
		}).
		DoFinally(func(ctx context.Context, st rx.SignalType) {
			msg.Release()
		})
}

func (*mustFailedSocket) RequestStream(msg payload.Payload) rx.Flux {
	return rx.
		NewFlux(func(ctx context.Context, producer rx.Producer) {
			producer.Error(errMustFailed)
		}).
		DoFinally(func(ctx context.Context, st rx.SignalType) {
			msg.Release()
		})
}

func (*mustFailedSocket) RequestChannel(msgs rx.Publisher) rx.Flux {
	return rx.
		NewFlux(func(ctx context.Context, producer rx.Producer) {
			producer.Error(errMustFailed)
		}).
		DoFinally(func(ctx context.Context, st rx.SignalType) {
			rx.ToFlux(msgs).
				DoOnSubscribe(func(ctx context.Context, s rx.Subscription) {
					s.Cancel()
				}).
				DoAfterNext(func(ctx context.Context, elem payload.Payload) {
					elem.Release()
				}).
				Subscribe(context.Background())
		})
}

func newBalancer(first *socketSupplier, others ...*socketSupplier) *balancer {
	return &balancer{
		mu:         &sync.Mutex{},
		actives:    make([]*availabilitySocket, 0),
		suppliers:  newSocketPool(first, others...),
		minActives: defaultMinActives,
		maxActives: defaultMaxActives,
	}
}

func newBalancerStarter(bu *implClientBuilder, uris []string) *balancerStarter {
	return &balancerStarter{
		bu:   bu,
		uris: uris,
	}
}

type balancerStarter struct {
	bu   *implClientBuilder
	uris []string
}

func (p *balancerStarter) Start() (ClientSocket, error) {
	if len(p.uris) < 2 {
		return nil, errAtLeastTwoSuppliers
	}
	suppliers := make([]*socketSupplier, 0)
	for _, uri := range p.uris {
		suppliers = append(suppliers, newSocketSupplier(p.bu, uri))
	}
	return newBalancer(suppliers[0], suppliers[1:]...), nil
}
