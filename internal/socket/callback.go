package socket

import (
	"io"

	"github.com/jjeffcaii/reactor-go"
	"github.com/rsocket/rsocket-go/internal/common"
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
	if closer, ok := s.pc.Raw().(io.Closer); ok {
		_ = closer.Close()
	}
	s.pc.Error(err)
}

type requestResponseCallback struct {
	sink  mono.Sink
	cache interface{}
}

func (s requestResponseCallback) stopWithError(err error) {
	s.sink.Error(err)
	common.TryRelease(s.cache)
}

type requestChannelCallback struct {
	snd rx.Subscription
	rcv flux.Processor
}

func (s requestChannelCallback) stopWithError(err error) {
	s.snd.Cancel()
	if closer, ok := s.rcv.Raw().(io.Closer); ok {
		_ = closer.Close()
	}
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

type respondChannelCallback struct {
	snd rx.Subscription
	rcv flux.Processor
}

func (s respondChannelCallback) stopWithError(err error) {
	if closer, ok := s.rcv.Raw().(io.Closer); ok {
		_ = closer.Close()
	}
	s.rcv.Error(err)
	s.snd.Cancel()
}
