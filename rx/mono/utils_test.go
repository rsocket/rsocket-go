package mono_test

import (
	"context"
	"testing"

	rsMono "github.com/jjeffcaii/reactor-go/mono"
	"github.com/jjeffcaii/reactor-go/scheduler"
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx/mono"
	"github.com/stretchr/testify/assert"
)

func TestCreateFromChannel(t *testing.T) {
	payloads := make(chan payload.Payload)
	err := make(chan error)

	go func() {
		defer close(payloads)
		defer close(err)
		p := payload.NewString("data", "metadata")
		payloads <- p
	}()

	background := context.Background()
	last, e := mono.CreateFromChannel(payloads, err).Block(background)
	if e != nil {
		t.Error(e)
	}

	assert.Equal(t, "data", last.DataUTF8())

	m, _ := last.MetadataUTF8()
	assert.Equal(t, "metadata", m)
}

func TestCreateFromChannelAndEmitError(t *testing.T) {
	payloads := make(chan payload.Payload)
	err := make(chan error)

	go func() {
		defer close(payloads)
		defer close(err)
		err <- errors.New("boom")
	}()
	_, e := mono.CreateFromChannel(payloads, err).Block(context.Background())
	assert.Error(t, e, "should emit error")
}

func TestCreateFromChannelWithNoEmitsOrErrors(t *testing.T) {
	payloads := make(chan payload.Payload)
	err := make(chan error)

	go func() {
		defer close(payloads)
		defer close(err)
	}()
	p, e := mono.CreateFromChannel(payloads, err).Block(context.Background())
	assert.Nil(t, p, "should be nil payload")
	assert.NoError(t, e, "should never emit error")
}

func TestToChannel(t *testing.T) {
	payloads := make(chan payload.Payload)
	err := make(chan error)

	go func() {
		defer close(payloads)
		defer close(err)
		p := payload.NewString("data", "metadata")
		payloads <- p
	}()

	valueChan, errChan := mono.CreateFromChannel(payloads, err).ToChan(context.Background())

loop:
	for {
		select {
		case p, ok := <-valueChan:
			if !ok {
				break loop
			}
			assert.Equal(t, "data", p.DataUTF8())
			md, _ := p.MetadataUTF8()
			assert.Equal(t, "metadata", md)
		case err := <-errChan:
			if err != nil {
				assert.NoError(t, err)
			}
			break loop
		}
	}

}

func TestToChannelEmitError(t *testing.T) {
	payloads := make(chan payload.Payload)
	err := make(chan error)

	go func() {
		defer close(payloads)
		defer close(err)

		for i := 1; i <= 10; i++ {
			err <- errors.New("boom!")
		}
	}()

	valueChan, errChan := mono.CreateFromChannel(payloads, err).ToChan(context.Background())

loop:
	for {
		select {
		case _, ok := <-valueChan:
			if !ok {
				break loop
			}
			assert.Fail(t, "should never receive anything")
		case err := <-errChan:
			if err != nil {
				break loop
			}
			assert.Fail(t, "should receive an error")
		}
	}

}

func TestRaw(t *testing.T) {
	fakePayload := payload.NewString("fake", "payload")
	res, err := mono.Raw(rsMono.Just(fakePayload)).Block(context.Background())
	assert.NoError(t, err)
	assert.True(t, payload.Equal(fakePayload, res))
}

func TestSubscribeWithChan(t *testing.T) {
	valueChan := make(chan payload.Payload)
	errChan := make(chan error)

	fakePayload := payload.NewString("fake data", "fake metadata")

	mono.
		Create(func(ctx context.Context, sink mono.Sink) {
			sink.Success(fakePayload)
		}).
		SubscribeOn(scheduler.Parallel()).
		SubscribeWithChan(context.Background(), valueChan, errChan)
	select {
	case next := <-valueChan:
		assert.True(t, payload.Equal(fakePayload, next), "result doesn't match")
	case err := <-errChan:
		assert.NoError(t, err, "should not return error")
	}
}
