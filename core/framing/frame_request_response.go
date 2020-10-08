package framing

import (
	"io"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

// RequestResponseFrame is RequestResponse frame.
type RequestResponseFrame struct {
	*baseDefaultFrame
}

// WriteableRequestResponseFrame is writeable RequestResponse frame.
type WriteableRequestResponseFrame struct {
	baseWriteableFrame
	metadata []byte
	data     []byte
}

// Validate returns error if frame is invalid.
func (r *RequestResponseFrame) Validate() (err error) {
	if r.header.Flag().Check(core.FlagMetadata) && r.body.Len() < 3 {
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

// WriteTo writes frame to writer.
func (r WriteableRequestResponseFrame) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int64
	wrote, err = r.header.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote
	wrote, err = writePayload(w, r.data, r.metadata)
	if err == nil {
		n += wrote
	}
	return
}

// Len returns length of frame.
func (r WriteableRequestResponseFrame) Len() int {
	return CalcPayloadFrameSize(r.data, r.metadata)
}

// NewWriteableRequestResponseFrame returns a new WriteableRequestResponseFrame.
func NewWriteableRequestResponseFrame(id uint32, data, metadata []byte, fg core.FrameFlag) core.WriteableFrame {
	if len(metadata) > 0 {
		fg |= core.FlagMetadata
	}
	return WriteableRequestResponseFrame{
		baseWriteableFrame: newBaseWriteableFrame(core.NewFrameHeader(id, core.FrameTypeRequestResponse, fg)),
		metadata:           metadata,
		data:               data,
	}
}

// NewRequestResponseFrame returns a new RequestResponseFrame.
func NewRequestResponseFrame(id uint32, data, metadata []byte, fg core.FrameFlag) *RequestResponseFrame {
	b := common.BorrowByteBuff()
	if len(metadata) > 0 {
		fg |= core.FlagMetadata
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
		newBaseDefaultFrame(core.NewFrameHeader(id, core.FrameTypeRequestResponse, fg), b),
	}
}
