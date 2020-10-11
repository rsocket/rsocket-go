package framing

import (
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
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
		if err := b.WriteUint24(len(metadata)); err != nil {
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
	if r.Header().Flag().Check(core.FlagMetadata) && r.bodyLen() < 3 {
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
		metadata = string(raw)
	}
	return
}

// DataUTF8 returns data as UTF8 string.
func (r *RequestResponseFrame) DataUTF8() string {
	return string(r.Data())
}
