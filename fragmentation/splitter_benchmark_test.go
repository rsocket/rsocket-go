package fragmentation

import (
	"strings"
	"testing"

	"github.com/rsocket/rsocket-go/common"
	"github.com/rsocket/rsocket-go/framing"
)

func BenchmarkToFragments(b *testing.B) {
	// 4m data + 1m metadata, 128k per fragment
	splitter, _ := NewSplitter(128)
	data := []byte(strings.Repeat("d", 4*1024*1024))
	metadata := []byte(strings.Repeat("m", 1024*1024))
	//h := framing.NewFrameHeader(77778888, framing.FrameTypePayload, framing.FlagMetadata, framing.FlagComplete)
	fn := func(idx int, fg framing.FrameFlag, body *common.ByteBuff) {
		common.ReturnByteBuffer(body)
	}
	for i := 0; i < b.N; i++ {
		err := splitter.Split(0, data, metadata, fn)
		if err != nil {
			b.Error(err)
		}
	}
}
