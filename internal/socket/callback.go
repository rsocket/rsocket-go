package socket

import (
	"github.com/jjeffcaii/reactor-go"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/rsocket/rsocket-go/rx/mono"
)

type callback interface {
	stopWithError(error)
}

type requestStreamCallback struct {
	pc flux.Processor
}

func (s requestStreamCallback) stopWithError(err error) {
	s.pc.Error(err)
}

type requestResponseCallback struct {
	pc mono.Processor
}

func (s requestResponseCallback) stopWithError(err error) {
	s.pc.Error(err)
}

type requestChannelCallback struct {
	snd rx.Subscription
	rcv flux.Processor
}

func (s requestChannelCallback) stopWithError(err error) {
	s.snd.Cancel()
	s.rcv.Error(err)
}

type requestResponseCallbackReverse struct {
	su reactor.Subscription
}

func (s requestResponseCallbackReverse) stopWithError(err error) {
	s.su.Cancel()
	// TODO: fill err
}

type requestStreamCallbackReverse struct {
	su rx.Subscription
}

func (s requestStreamCallbackReverse) stopWithError(err error) {
	s.su.Cancel()
	// TODO: fill error
}

type requestChannelCallbackReverse struct {
	snd rx.Subscription
	rcv flux.Processor
}

func (s requestChannelCallbackReverse) stopWithError(err error) {
	s.rcv.Error(err)
	s.snd.Cancel()
}
