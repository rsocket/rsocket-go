package mono

import (
	"context"
	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx/flux"
	"github.com/stretchr/testify/assert"
	"testing"
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
	last, e :=
		CreateFromChannel(payloads, err).
		Block(background)
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

	background := context.Background()
	_, e :=
		CreateFromChannel(payloads, err).
			Block(background)

	if e == nil {
		t.Fail()
	}
}

func TestCreateFromChannelWithNoEmitsOrErrors(t *testing.T) {
	payloads := make(chan payload.Payload)
	err := make(chan error)

	go func() {
		defer close(payloads)
		defer close(err)
	}()

	background := context.Background()
	_, e :=
		CreateFromChannel(payloads, err).
			Block(background)

	if e != nil {
		t.Fail()
	}
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

	f := flux.CreateFromChannel(payloads, err)

	channel, chanerrors := flux.ToChannel(f, context.Background())

loop:
	for {
		select {
		case p, o := <-channel:
			if o {
				assert.Equal(t, "data", p.DataUTF8())
				md, _ := p.MetadataUTF8()
				assert.Equal(t, "metadata", md)
			} else {
				break loop
			}
		case err := <-chanerrors:
			if err != nil {
				t.Error(err)
				break loop
			}
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

	f := flux.CreateFromChannel(payloads, err)

	channel, chanerrors := flux.ToChannel(f, context.Background())

loop:
	for {
		select {
		case _, o := <-channel:
			if o {
				t.Fail()
			} else {
				break loop
			}
		case err := <-chanerrors:
			if err != nil {
				break loop
			} else {
				t.Fail()
			}
		}
	}

}
