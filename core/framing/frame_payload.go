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

func (p *PayloadFrame) MustMetadataUTF8() string {
	s, ok := p.MetadataUTF8()
	if !ok {
		panic("cannot convert metadata to utf8")
	}
	return s
}

// DataUTF8 returns data as UTF8 string.
func (p *PayloadFrame) DataUTF8() string {
	return string(p.Data())
}

type PayloadFrameSupport struct {
	*tinyFrame
	metadata []byte
	data     []byte
}

func (p PayloadFrameSupport) DataUTF8() string {
	return string(p.data)
}

func (p PayloadFrameSupport) MetadataUTF8() (metadata string, ok bool) {
	if p.header.Flag().Check(core.FlagMetadata) {
		metadata = string(p.metadata)
		ok = true
	}
	return
}

func (p PayloadFrameSupport) Data() []byte {
	return p.data
}

func (p PayloadFrameSupport) Metadata() ([]byte, bool) {
	return p.metadata, p.header.Flag().Check(core.FlagMetadata)
}

func (p PayloadFrameSupport) WriteTo(w io.Writer) (n int64, err error) {
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

func (p PayloadFrameSupport) Len() int {
	return CalcPayloadFrameSize(p.data, p.metadata)
}

// NewPayloadFrameSupport returns a new payload frame.
func NewPayloadFrameSupport(id uint32, data, metadata []byte, flag core.FrameFlag) *PayloadFrameSupport {
	if len(metadata) > 0 {
		flag |= core.FlagMetadata
	}
	h := core.NewFrameHeader(id, core.FrameTypePayload, flag)
	t := newTinyFrame(h)
	return &PayloadFrameSupport{
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
