package socket_test

import (
	"math"
	"testing"

	"github.com/rsocket/rsocket-go/internal/socket"
	"github.com/rsocket/rsocket-go/rx"
	"github.com/stretchr/testify/assert"
)

func TestToUint32RequestN(t *testing.T) {
	assert.Equal(t, uint32(1), socket.ToUint32RequestN(1))
	assert.Panics(t, func() {
		socket.ToUint32RequestN(-1)
	}, "should panic")
	assert.Equal(t, uint32(rx.RequestMax), socket.ToUint32RequestN(math.MaxInt64))
}

func TestToIntRequestN(t *testing.T) {
	assert.Equal(t, 1, socket.ToIntRequestN(1))
	assert.Equal(t, rx.RequestMax, socket.ToIntRequestN(math.MaxUint32))
}
