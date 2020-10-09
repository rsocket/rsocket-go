package mono_test

import (
	"context"
	"testing"

	"github.com/rsocket/rsocket-go/payload"
	"github.com/rsocket/rsocket-go/rx/mono"
)

var _fakePayload = payload.NewString("", "")

func BenchmarkDefaultProxy(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mono.Just(_fakePayload).Subscribe(context.Background())
		}
	})
}

func BenchmarkOneshotProxy(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mono.JustOneshot(_fakePayload).Subscribe(context.Background())
		}
	})
}
