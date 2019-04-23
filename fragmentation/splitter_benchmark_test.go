package fragmentation

import (
	"testing"

	"github.com/rsocket/rsocket-go/common"
	"github.com/rsocket/rsocket-go/framing"
)

func BenchmarkToFragments(b *testing.B) {
	// 4m data + 1m metadata, 128k per fragment
	splitter, _ := NewSplitter(128)
	data := []byte(common.RandAlphanumeric(4 * 1024 * 1024))
	metadata := []byte(common.RandAlphanumeric(1024 * 1024))

	fn := func(idx int, fg framing.FrameFlag, body *common.ByteBuff) {
		common.ReturnByteBuffer(body)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := splitter.Split(0, data, metadata, fn)
		if err != nil {
			b.Error(err)
		}
	}
	b.StopTimer()
}
