package fragmentation

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/stretchr/testify/assert"
)

func TestFragmentPayload(t *testing.T) {
	const totals = 10
	const sid = uint32(1)
	dataSb := &strings.Builder{}
	metadataSb := &strings.Builder{}
	fr := NewJoiner(framing.NewPayloadFrame(sid, []byte("(ROOT)"), []byte("(ROOT)"), core.FlagFollow|core.FlagMetadata|core.FlagNext))
	dataSb.WriteString("(ROOT)")
	metadataSb.WriteString("(ROOT)")
	for i := 0; i < totals; i++ {
		data := fmt.Sprintf("(data%04d)", i)
		dataSb.WriteString(data)
		var frame *framing.PayloadFrame
		if i < 3 {
			metadata := fmt.Sprintf("(meta%04d)", i)
			metadataSb.WriteString(metadata)
			frame = framing.NewPayloadFrame(sid, []byte(data), []byte(metadata), core.FlagFollow|core.FlagMetadata)
		} else if i != totals-1 {
			frame = framing.NewPayloadFrame(sid, []byte(data), nil, core.FlagFollow)
		} else {
			frame = framing.NewPayloadFrame(sid, []byte(data), nil, 0)
		}
		fr.Push(frame)
	}
	m, _ := fr.MetadataUTF8()
	assert.Equal(t, sid, fr.Header().StreamID(), "stream id doesn't match")
	assert.Equal(t, core.FlagFollow|core.FlagMetadata|core.FlagNext, fr.Header().Flag(), "flag doesn't match")
	assert.Equal(t, metadataSb.String(), m, "metadata doesn't match")
	assert.Equal(t, dataSb.String(), fr.DataUTF8(), "data doesn't match")
}
