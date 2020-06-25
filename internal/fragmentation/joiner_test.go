package fragmentation

import (
	"fmt"
	"log"
	"testing"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
)

func TestFragmentPayload(t *testing.T) {
	const totals = 10
	const sid = uint32(1)
	fr := NewJoiner(framing.NewPayloadFrame(sid, []byte("(ROOT)"), []byte("(ROOT)"), core.FlagFollow|core.FlagMetadata))
	for i := 0; i < totals; i++ {
		data := fmt.Sprintf("(data%04d)", i)
		var frame *framing.PayloadFrame
		if i < 3 {
			meta := fmt.Sprintf("(meta%04d)", i)
			frame = framing.NewPayloadFrame(sid, []byte(data), []byte(meta), core.FlagFollow|core.FlagMetadata)
		} else if i != totals-1 {
			frame = framing.NewPayloadFrame(sid, []byte(data), nil, core.FlagFollow)
		} else {
			frame = framing.NewPayloadFrame(sid, []byte(data), nil, 0)
		}
		fr.Push(frame)
	}
	m, _ := fr.MetadataUTF8()
	log.Println("header:", fr.Header())
	log.Println("metadata:", m)
	log.Println("data:", fr.DataUTF8())
}
