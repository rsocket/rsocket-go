package common

import "testing"

func BenchmarkNewUint24(b *testing.B) {
	n := RandIntn(MaxUint24)
	for i := 0; i < b.N; i++ {
		_ = NewUint24(n)
	}
}

func BenchmarkUint24_AsInt(b *testing.B) {
	n := NewUint24(RandIntn(MaxUint24))
	for i := 0; i < b.N; i++ {
		_ = n.AsInt()
	}
}
