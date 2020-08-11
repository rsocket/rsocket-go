package common_test

import (
	"testing"

	. "github.com/rsocket/rsocket-go/internal/common"
	"github.com/stretchr/testify/assert"
)

func BenchmarkNewUint24(b *testing.B) {
	n := RandIntn(MaxUint24)
	for i := 0; i < b.N; i++ {
		_ = MustNewUint24(n)
	}
}

func BenchmarkUint24_AsInt(b *testing.B) {
	n := MustNewUint24(RandIntn(MaxUint24))
	for i := 0; i < b.N; i++ {
		_ = n.AsInt()
	}
}

func TestMustNewUint24(t *testing.T) {
	func() {
		defer func() {
			e := recover()
			assert.True(t, IsExceedMaximumUint24Error(e.(error)), "should failed")
		}()
		_ = MustNewUint24(MaxUint24 + 1)
	}()
	func() {
		defer func() {
			e := recover()
			assert.True(t, IsNegativeUint24Error(e.(error)), "should failed")
		}()
		_ = MustNewUint24(-1)
	}()
}

func TestUint24(t *testing.T) {
	testSingle(t, 0)
	for range [1_000_000]struct{}{} {
		testSingle(t, RandIntn(MaxUint24))
	}
	testSingle(t, MaxUint24)
	// negative
	_, err := NewUint24(-1)
	assert.Error(t, err, "negative number should failed")

	// over maximum number
	_, err = NewUint24(MaxUint24 + 1)
	assert.Error(t, err, "over maximum number should failed")
}

func TestUint24_WriteTo(t *testing.T) {
	for _, n := range []int{0, 1, RandIntn(MaxUint24), MaxUint24} {
		v := MustNewUint24(n)
		b := NewByteBuff()
		wrote, err := v.WriteTo(b)
		assert.NoError(t, err, "write uint24 failed")
		assert.Equal(t, int64(3), wrote, "wrote bytes length should be 3")
		assert.Equal(t, n, NewUint24Bytes(b.Bytes()).AsInt(), "bad uint24 result")
	}
}

func testSingle(t *testing.T, n int) {
	x := MustNewUint24(n)
	assert.Equal(t, n, x.AsInt(), "bad new from int")
	y := NewUint24Bytes(x.Bytes())
	assert.Equal(t, n, y.AsInt(), "bad new from bytes")
}
