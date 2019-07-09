package socket

import (
	rs "github.com/jjeffcaii/reactor-go"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/rsocket/rsocket-go/rx/mono"
)

type reqRS struct {
	pc flux.Processor
}

func (s reqRS) Close() error {
	s.pc.Error(errSocketClosed)
	return nil
}

type reqRR struct {
	pc mono.Processor
}

func (s reqRR) Close() error {
	s.pc.Error(errSocketClosed)
	return nil
}

type reqRC struct {
	snd rx.Subscription
	rcv flux.Processor
}

func (s reqRC) Close() error {
	s.snd.Cancel()
	s.rcv.Error(errSocketClosed)
	return nil
}

type resRR struct {
	su rs.Subscription
}

func (s resRR) Close() error {
	s.su.Cancel()
	return nil
}

type resRS struct {
	su rx.Subscription
}

func (s resRS) Close() error {
	s.su.Cancel()
	return nil
}

type resRC struct {
	snd rx.Subscription
	rcv flux.Processor
}

func (s resRC) Close() error {
	s.rcv.Error(errSocketClosed)
	s.snd.Cancel()
	return nil
}
