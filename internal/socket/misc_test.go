package socket

import (
	"math"
	"testing"

	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/stretchr/testify/assert"
)

func TestToUint32RequestN(t *testing.T) {
	assert.Equal(t, uint32(1), ToUint32RequestN(1))
	assert.Panics(t, func() {
		ToUint32RequestN(-1)
	}, "should panic")
	assert.Equal(t, uint32(rx.RequestMax), ToUint32RequestN(math.MaxInt64))
}

func TestToIntRequestN(t *testing.T) {
	assert.Equal(t, 1, ToIntRequestN(1))
	assert.Equal(t, rx.RequestMax, ToIntRequestN(math.MaxUint32))
}

func TestTryRecover(t *testing.T) {
	assert.NoError(t, tryRecover(nil))
	e := errors.New("fake error")
	assert.Equal(t, e, tryRecover(e))
	assert.Error(t, e, tryRecover("fake error"))
	assert.Error(t, e, tryRecover(struct{}{}))
}
