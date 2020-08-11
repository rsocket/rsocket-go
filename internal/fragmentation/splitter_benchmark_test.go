package fragmentation

import (
	"testing"

	"github.com/rsocket/rsocket-go/internal/common"
)

func BenchmarkToFragments(b *testing.B) {
	// 4m data + 1m metadata, 128k per fragment
	mtu := 128
	data := []byte(common.RandAlphanumeric(4 * 1024 * 1024))
	metadata := []byte(common.RandAlphanumeric(1024 * 1024))

	fn := func(idx int, result SplitResult) {
	}
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Split(mtu, data, metadata, fn)
		}
	})
	b.StopTimer()
}
