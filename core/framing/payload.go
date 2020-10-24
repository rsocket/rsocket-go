package framing

import (
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/bytesconv"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/u24"
)

// PayloadFrame is payload frame.
type PayloadFrame struct {
	*bufferedFrame
}

// NewPayloadFrame returns a new PayloadFrame.
func NewPayloadFrame(id uint32, data, metadata []byte, flag core.FrameFlag) *PayloadFrame {
	if len(metadata) > 0 {
		flag |= core.FlagMetadata
	}

	bb := common.BorrowByteBuff()

	if err := core.WriteFrameHeader(bb, id, core.FrameTypePayload, flag); err != nil {
		common.ReturnByteBuff(bb)
		panic(err)
	}

	if len(metadata) > 0 {
		if err := u24.WriteUint24(bb, len(metadata)); err != nil {
			common.ReturnByteBuff(bb)
			panic(err)
		}
		if _, err := bb.Write(metadata); err != nil {
			common.ReturnByteBuff(bb)
			panic(err)
		}
	}
	if len(data) > 0 {
		if _, err := bb.Write(data); err != nil {
			common.ReturnByteBuff(bb)
			panic(err)
		}
	}

	return &PayloadFrame{
		bufferedFrame: newBufferedFrame(bb),
	}
}

// Validate returns error if frame is invalid.
func (p *PayloadFrame) Validate() (err error) {
	// Minimal length should be 3 if metadata exists.
	if p.HasFlag(core.FlagMetadata) && p.bodyLen() < 3 {
		err = errIncompleteFrame
	}
	return
}

// Metadata returns metadata bytes.
func (p *PayloadFrame) Metadata() ([]byte, bool) {
	return p.trySliceMetadata(0)
}

// Data returns data bytes.
func (p *PayloadFrame) Data() []byte {
	return p.trySliceData(0)
}

// MetadataUTF8 returns metadata as UTF8 string.
func (p *PayloadFrame) MetadataUTF8() (metadata string, ok bool) {
	raw, ok := p.Metadata()
	if ok {
		metadata = bytesconv.BytesToString(raw)
	}
	return
}

// DataUTF8 returns data as UTF8 string.
func (p *PayloadFrame) DataUTF8() (data string) {
	d := p.Data()
	if len(d) > 0 {
		data = bytesconv.BytesToString(d)
	}
	return
}
