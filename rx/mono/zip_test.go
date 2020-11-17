package mono_test

import (
	"context"
	"testing"

	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/rsocket/rsocket-go/rx/mono"
	"github.com/stretchr/testify/assert"
)

func TestZipBuilder_ToMono(t *testing.T) {
	v, err := mono.Zip(mono.Just(_fakePayload), mono.Just(_fakePayload)).
		ToMono(func(tuple rx.Tuple) (payload.Payload, error) {
			assert.Equal(t, 2, tuple.Len())
			for i := 0; i < tuple.Len(); i++ {
				v, err := tuple.Get(i)
				assert.NoError(t, err)
				assert.Equal(t, _fakePayload, v)
			}
			return _fakePayload, nil
		}).
		Block(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, _fakePayload, v)
}

func TestZip_Empty(t *testing.T) {
	assert.Panics(t, func() {
		mono.ZipAll().ToMono(func(tuple rx.Tuple) (payload.Payload, error) {
			return _fakePayload, nil
		}).Block(context.Background())
	})
}
