package socket

import (
	rs "github.com/jjeffcaii/reactor-go"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/rsocket/rsocket-go/rx/mono"
)

type closerWithError interface {
	Close(error)
}

type reqRS struct {
	pc flux.Processor
}

func (s reqRS) Close(err error) {
	s.pc.Error(err)
}

type reqRR struct {
	pc mono.Processor
}

func (s reqRR) Close(err error) {
	s.pc.Error(err)
}

type reqRC struct {
	snd rx.Subscription
	rcv flux.Processor
}

func (s reqRC) Close(err error) {
	s.snd.Cancel()
	s.rcv.Error(err)
}

type resRR struct {
	su rs.Subscription
}

func (s resRR) Close(err error) {
	s.su.Cancel()
	// TODO: fill err
}

type resRS struct {
	su rx.Subscription
}

func (s resRS) Close(err error) {
	s.su.Cancel()
	// TODO: fill error
}

type resRC struct {
	snd rx.Subscription
	rcv flux.Processor
}

func (s resRC) Close(err error) {
	s.rcv.Error(err)
	s.snd.Cancel()
}
