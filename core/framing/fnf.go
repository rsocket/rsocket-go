package framing

import (
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/bytesconv"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/u24"
)

// FireAndForgetFrame is FireAndForget frame.
type FireAndForgetFrame struct {
	*bufferedFrame
}

// NewFireAndForgetFrame returns a new FireAndForgetFrame.
func NewFireAndForgetFrame(sid uint32, data, metadata []byte, flag core.FrameFlag) *FireAndForgetFrame {
	if len(metadata) > 0 {
		flag |= core.FlagMetadata
	}

	bb := common.BorrowByteBuff()

	if err := core.WriteFrameHeader(bb, sid, core.FrameTypeRequestFNF, flag); err != nil {
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
	if _, err := bb.Write(data); err != nil {
		common.ReturnByteBuff(bb)
		panic(err)
	}

	return &FireAndForgetFrame{
		bufferedFrame: newBufferedFrame(bb),
	}
}

// Validate returns error if frame is invalid.
func (f *FireAndForgetFrame) Validate() (err error) {
	if f.HasFlag(core.FlagMetadata) && f.bodyLen() < 3 {
		err = errIncompleteFrame
	}
	return
}

// Metadata returns metadata bytes.
func (f *FireAndForgetFrame) Metadata() ([]byte, bool) {
	return f.trySliceMetadata(0)
}

// Data returns data bytes.
func (f *FireAndForgetFrame) Data() []byte {
	return f.trySliceData(0)
}

// MetadataUTF8 returns metadata as UTF8 string.
func (f *FireAndForgetFrame) MetadataUTF8() (metadata string, ok bool) {
	raw, ok := f.Metadata()
	if ok {
		metadata = bytesconv.BytesToString(raw)
	}
	return
}

// DataUTF8 returns data as UTF8 string.
func (f *FireAndForgetFrame) DataUTF8() (data string) {
	b := f.Data()
	if len(b) > 0 {
		data = bytesconv.BytesToString(b)
	}
	return
}
