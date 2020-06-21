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
	fn := func(idx int, result SplitResult) {
		sid := uint32(77778888)
		if idx == 0 {
			f := framing.NewPayloadFrameSupport(sid, result.Data, result.Metadata, framing.FlagComplete|result.Flag)
			joiner = NewJoiner(f)
		} else {
			f := framing.NewPayloadFrameSupport(sid, result.Data, result.Metadata, result.Flag)
			joiner.Push(f)
		}
	}
	Split(mtu, data, metadata, fn)
	return
}
