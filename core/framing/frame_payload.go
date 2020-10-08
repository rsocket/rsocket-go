package framing

import (
	"io"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

// PayloadFrame is payload frame.
type PayloadFrame struct {
	*baseDefaultFrame
}

// WriteablePayloadFrame is writeable Payload frame.
type WriteablePayloadFrame struct {
	baseWriteableFrame
	metadata []byte
	data     []byte
}

// NewWriteablePayloadFrame returns a new WriteablePayloadFrame.
func NewWriteablePayloadFrame(id uint32, data, metadata []byte, flag core.FrameFlag) *WriteablePayloadFrame {
	if len(metadata) > 0 {
		flag |= core.FlagMetadata
	}
	h := core.NewFrameHeader(id, core.FrameTypePayload, flag)
	t := newBaseWriteableFrame(h)
	return &WriteablePayloadFrame{
		baseWriteableFrame: t,
		metadata:           metadata,
		data:               data,
	}
}

// NewPayloadFrame returns a new PayloadFrame.
func NewPayloadFrame(id uint32, data, metadata []byte, flag core.FrameFlag) *PayloadFrame {
	b := common.BorrowByteBuff()
	if len(metadata) > 0 {
		flag |= core.FlagMetadata
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
	return &PayloadFrame{
		newBaseDefaultFrame(core.NewFrameHeader(id, core.FrameTypePayload, flag), b),
	}
}

// Validate returns error if frame is invalid.
func (p *PayloadFrame) Validate() (err error) {
	// Minimal length should be 3 if metadata exists.
	if p.header.Flag().Check(core.FlagMetadata) && p.body.Len() < 3 {
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
		metadata = string(raw)
	}
	return
}

// DataUTF8 returns data as UTF8 string.
func (p *PayloadFrame) DataUTF8() string {
	return string(p.Data())
}

// Data returns data bytes.
func (p WriteablePayloadFrame) Data() []byte {
	return p.data
}

// Metadata returns metadata bytes.
func (p WriteablePayloadFrame) Metadata() (metadata []byte, ok bool) {
	ok = p.header.Flag().Check(core.FlagMetadata)
	if ok {
		metadata = p.metadata
	}
	return
}

// DataUTF8 returns data as UTF8 string.
func (p WriteablePayloadFrame) DataUTF8() (data string) {
	if p.data != nil {
		data = string(p.data)
	}
	return
}

// MetadataUTF8 returns metadata as UTF8 string.
func (p WriteablePayloadFrame) MetadataUTF8() (metadata string, ok bool) {
	ok = p.header.Flag().Check(core.FlagMetadata)
	if ok {
		metadata = string(p.metadata)
	}
	return
}

// WriteTo writes frame to writer.
func (p WriteablePayloadFrame) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int64
	wrote, err = p.header.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote
	wrote, err = writePayload(w, p.data, p.metadata)
	if err == nil {
		n += wrote
	}
	return
}

// Len returns length of frame.
func (p WriteablePayloadFrame) Len() int {
	return CalcPayloadFrameSize(p.data, p.metadata)
}
