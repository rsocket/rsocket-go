package mono_test

import (
	"context"
	"testing"
	"time"

	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx/mono"
	"github.com/stretchr/testify/assert"
)

func TestDelay(t *testing.T) {
	begin := time.Now()
	delay := 100 * time.Millisecond
	v, err := mono.Delay(delay).
		ToMono(func() (payload.Payload, error) {
			return _fakePayload, nil
		}).
		Block(context.Background())
	assert.NoError(t, err, "should not return error")
	assert.Equal(t, _fakePayload, v, "value doesn't match")
	elapsed := time.Since(begin)
	assert.Greater(t, int64(elapsed), int64(delay), "should be greater than delay duration")
}
