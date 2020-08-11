package framing

import (
	"io"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

// PayloadFrame is payload frame.
type PayloadFrame struct {
	*RawFrame
}

type WriteablePayloadFrame struct {
	*tinyFrame
	metadata []byte
	data     []byte
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

func (p *PayloadFrame) MetadataUTF8() (metadata string, ok bool) {
	raw, ok := p.Metadata()
	if ok {
		metadata = string(raw)
	}
	return
}

func (p *PayloadFrame) DataUTF8() string {
	return string(p.Data())
}

func (p WriteablePayloadFrame) Data() []byte {
	return p.data
}

func (p WriteablePayloadFrame) Metadata() (metadata []byte, ok bool) {
	ok = p.header.Flag().Check(core.FlagMetadata)
	if ok {
		metadata = p.metadata
	}
	return
}

func (p WriteablePayloadFrame) DataUTF8() (data string) {
	if p.data != nil {
		data = string(p.data)
	}
	return
}

func (p WriteablePayloadFrame) MetadataUTF8() (metadata string, ok bool) {
	ok = p.header.Flag().Check(core.FlagMetadata)
	if ok {
		metadata = string(p.metadata)
	}
	return
}

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

func (p WriteablePayloadFrame) Len() int {
	return CalcPayloadFrameSize(p.data, p.metadata)
}

// NewWriteablePayloadFrame returns a new payload frame.
func NewWriteablePayloadFrame(id uint32, data, metadata []byte, flag core.FrameFlag) *WriteablePayloadFrame {
	if len(metadata) > 0 {
		flag |= core.FlagMetadata
	}
	h := core.NewFrameHeader(id, core.FrameTypePayload, flag)
	t := newTinyFrame(h)
	return &WriteablePayloadFrame{
		tinyFrame: t,
		metadata:  metadata,
		data:      data,
	}
}

// NewPayloadFrame returns a new payload frame.
func NewPayloadFrame(id uint32, data, metadata []byte, flag core.FrameFlag) *PayloadFrame {
	bf := common.NewByteBuff()
	if len(metadata) > 0 {
		flag |= core.FlagMetadata
		if err := bf.WriteUint24(len(metadata)); err != nil {
			panic(err)
		}
		if _, err := bf.Write(metadata); err != nil {
			panic(err)
		}
	}
	if len(data) > 0 {
		if _, err := bf.Write(data); err != nil {
			panic(err)
		}
	}
	return &PayloadFrame{
		NewRawFrame(core.NewFrameHeader(id, core.FrameTypePayload, flag), bf),
	}
}
