package fragmentation

import (
	"testing"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/framing"
	"github.com/stretchr/testify/assert"
)

func TestSplitter_Split(t *testing.T) {
	const mtu = 128
	data := []byte(common.RandAlphanumeric(1024))
	metadata := []byte(common.RandAlphanumeric(512))

	joiner, err := split2joiner(mtu, data, metadata)
	assert.NoError(t, err, "split failed")

	m, ok := joiner.Metadata()
	assert.True(t, ok, "bad metadata")
	assert.Equal(t, metadata, m, "bad metadata")
	assert.Equal(t, data, joiner.Data(), "bad data")
}

func split2joiner(mtu int, data, metadata []byte) (joiner Joiner, err error) {
	fn := func(idx int, fg framing.FrameFlag, body *common.ByteBuff) {
		if idx == 0 {
			h := framing.NewFrameHeader(77778888, framing.FrameTypePayload, framing.FlagComplete|fg)
			joiner = NewJoiner(&framing.PayloadFrame{
				RawFrame: framing.NewRawFrame(h, body),
			})
		} else {
			h := framing.NewFrameHeader(77778888, framing.FrameTypePayload, fg)
			joiner.Push(&framing.PayloadFrame{
				RawFrame: framing.NewRawFrame(h, body),
			})
		}
	}
	Split(mtu, data, metadata, fn)
	return
}
