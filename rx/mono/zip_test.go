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
		ToMono(func(items rx.Tuple) (payload.Payload, error) {
			assert.Equal(t, 2, items.Len())
			for i := 0; i < len(items); i++ {
				v, err := items.Get(i)
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
		_, _ = mono.ZipAll().ToMono(func(tuple rx.Tuple) (payload.Payload, error) {
			return _fakePayload, nil
		}).Block(context.Background())
	})
}

func TestZipWith(t *testing.T) {
	p1 := payload.NewString("hello", "")
	p2 := payload.NewString("world", "")
	res, err := mono.Just(p1).
		ZipWith(mono.Just(p2), func(first, second mono.Item) (payload.Payload, error) {
			data := first.V.DataUTF8() + " " + second.V.DataUTF8() + "!"
			return payload.NewString(data, ""), nil
		}).
		Block(context.Background())
	assert.NoError(t, err, "should not return error")
	assert.Equal(t, "hello world!", res.DataUTF8(), "bad result")
}
