package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUint24(t *testing.T) {
	n := RandIntn(MaxUint24)
	x := NewUint24(n)
	assert.Equal(t, n, x.AsInt(), "bad new from int")
	y := NewUint24Bytes(x.Bytes())
	assert.Equal(t, n, y.AsInt(), "bad new from bytes")
}
