package socket_test

import (
	"context"
	"testing"

	"github.com/rsocket/rsocket-go/internal/socket"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/rsocket/rsocket-go/rx/mono"
	"github.com/stretchr/testify/assert"
	"go.uber.org/atomic"
)

var emptyAbstractRSocket = &socket.AbstractRSocket{}
var fakeRequest = payload.New(fakeData, fakeMetadata)

func TestAbstractRSocket_FireAndForget(t *testing.T) {
	called := atomic.NewBool(false)
	s := &socket.AbstractRSocket{
		FF: func(payload payload.Payload) {
			called.CAS(false, true)
		},
	}
	s.FireAndForget(fakeRequest)
	assert.True(t, called.Load())

	assert.NotPanics(t, func() {
		emptyAbstractRSocket.FireAndForget(fakeRequest)
	})
}

func TestAbstractRSocket_RequestResponse(t *testing.T) {
	s := &socket.AbstractRSocket{
		RR: func(p payload.Payload) mono.Mono {
			return mono.Just(p)
		},
	}
	res, err := s.RequestResponse(fakeRequest).Block(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, fakeData, res.Data())
	assert.Equal(t, fakeMetadata, extractMetadata(res))

	_, err = emptyAbstractRSocket.RequestResponse(fakeRequest).Block(context.Background())
	assert.Error(t, err, "should return an error")
}

func TestAbstractRSocket_MetadataPush(t *testing.T) {
	called := atomic.NewBool(false)
	s := &socket.AbstractRSocket{
		MP: func(p payload.Payload) {
			called.CAS(false, true)
		},
	}

	assert.NotPanics(t, func() {
		s.MetadataPush(fakeRequest)
	})
	assert.NotPanics(t, func() {
		emptyAbstractRSocket.MetadataPush(fakeRequest)
	})
}

func TestAbstractRSocket_RequestStream(t *testing.T) {
	s := &socket.AbstractRSocket{
		RS: func(p payload.Payload) flux.Flux {
			return flux.Just(p)
		},
	}

	var res []payload.Payload

	_, err := s.RequestStream(fakeRequest).
		DoOnNext(func(input payload.Payload) error {
			res = append(res, input)
			return nil
		}).
		BlockLast(context.Background())
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, fakeRequest, res[0])

	_, err = emptyAbstractRSocket.RequestStream(fakeRequest).BlockLast(context.Background())
	assert.Error(t, err, "should return an error")
}

func TestAbstractRSocket_RequestChannel(t *testing.T) {
	s := &socket.AbstractRSocket{
		RC: func(initialRequest payload.Payload, publisher flux.Flux) flux.Flux {
			return flux.Clone(publisher)
		},
	}
	var res []payload.Payload
	_, err := s.RequestChannel(fakeRequest, flux.Just(fakeRequest)).
		DoOnNext(func(input payload.Payload) error {
			res = append(res, input)
			return nil
		}).
		BlockLast(context.Background())
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, fakeRequest, res[0])

	_, err = emptyAbstractRSocket.RequestChannel(fakeRequest, flux.Just(fakeRequest)).BlockFirst(context.Background())
	assert.Error(t, err, "should return an error")
}
