package framing

import (
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/bytesconv"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/u24"
)

// RequestResponseFrame is RequestResponse frame.
type RequestResponseFrame struct {
	*bufferedFrame
}

// NewRequestResponseFrame returns a new RequestResponseFrame.
func NewRequestResponseFrame(id uint32, data, metadata []byte, fg core.FrameFlag) *RequestResponseFrame {
	if len(metadata) > 0 {
		fg |= core.FlagMetadata
	}

	b := common.BorrowByteBuff()

	if err := core.WriteFrameHeader(b, id, core.FrameTypeRequestResponse, fg); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}

	if len(metadata) > 0 {
		if err := u24.WriteUint24(b, len(metadata)); err != nil {
			common.ReturnByteBuff(b)
			panic(err)
		}
		if _, err := b.Write(metadata); err != nil {
			common.ReturnByteBuff(b)
			panic(err)
		}
	}
	if len(data) > 0 {
		if _, err := b.Write(data); err != nil {
			common.ReturnByteBuff(b)
			panic(err)
		}
	}
	return &RequestResponseFrame{
		bufferedFrame: newBufferedFrame(b),
	}
}

// Validate returns error if frame is invalid.
func (r *RequestResponseFrame) Validate() (err error) {
	if r.HasFlag(core.FlagMetadata) && r.bodyLen() < 3 {
		err = errIncompleteFrame
	}
	return
}

// Metadata returns metadata bytes.
func (r *RequestResponseFrame) Metadata() ([]byte, bool) {
	return r.trySliceMetadata(0)
}

// Data returns data bytes.
func (r *RequestResponseFrame) Data() []byte {
	return r.trySliceData(0)
}

// MetadataUTF8 returns metadata as UTF8 string.
func (r *RequestResponseFrame) MetadataUTF8() (metadata string, ok bool) {
	raw, ok := r.Metadata()
	if ok {
		metadata = bytesconv.BytesToString(raw)
	}
	return
}

// DataUTF8 returns data as UTF8 string.
func (r *RequestResponseFrame) DataUTF8() (data string) {
	b := r.Data()
	if len(b) > 0 {
		data = bytesconv.BytesToString(b)
	}
	return
}
