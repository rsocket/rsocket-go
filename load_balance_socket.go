package rsocket

import (
	"context"
	"fmt"

	"github.com/rsocket/rsocket-go/common"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
)

type availabilitySocket struct {
	origin   ClientSocket
	supplier *socketSupplier
}

func (p *availabilitySocket) FireAndForget(msg payload.Payload) {
	p.origin.FireAndForget(msg)
}

func (p *availabilitySocket) MetadataPush(msg payload.Payload) {
	p.origin.MetadataPush(msg)
}

func (p *availabilitySocket) RequestResponse(msg payload.Payload) rx.Mono {
	return p.origin.RequestResponse(msg).DoFinally(p.notifyFinally)
}

func (p *availabilitySocket) RequestStream(msg payload.Payload) rx.Flux {
	return p.origin.RequestStream(msg).DoFinally(p.notifyFinally)
}

func (p *availabilitySocket) RequestChannel(msgs rx.Publisher) rx.Flux {
	return p.origin.RequestChannel(msgs).DoFinally(p.notifyFinally)
}

func (p *availabilitySocket) String() string {
	return fmt.Sprintf("Socket{%s}", p.supplier.u)
}

func (p *availabilitySocket) Close() error {
	return p.origin.Close()
}

func (p *availabilitySocket) availability() float64 {
	if common.NowInMicrosecond()-p.supplier.stamp > p.supplier.tau {
		p.supplier.updateErrorPercentage(1)
	}
	return p.supplier.errorPercentage.Value()
}

func (p *availabilitySocket) notifyFinally(ctx context.Context, st rx.SignalType) {
	switch st {
	case rx.SignalComplete:
		p.supplier.updateErrorPercentage(1)
	case rx.SignalError:
		p.supplier.errorPercentage.Insert(0)
	}
}

func newWeightedSocket(origin ClientSocket, supplier *socketSupplier) *availabilitySocket {
	return &availabilitySocket{
		origin:   origin,
		supplier: supplier,
	}
}
