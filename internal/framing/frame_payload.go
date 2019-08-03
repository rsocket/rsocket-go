package framing

import (
	"fmt"

	"github.com/rsocket/rsocket-go/internal/common"
)

// FramePayload is payload frame.
type FramePayload struct {
	*BaseFrame
}

// Validate returns error if frame is invalid.
func (p *FramePayload) Validate() (err error) {
	return
}

func (p *FramePayload) String() string {
	m, _ := p.MetadataUTF8()
	return fmt.Sprintf("FramePayload{%s,data=%s,metadata=%s}", p.header, p.DataUTF8(), m)
}

// Metadata returns metadata bytes.
func (p *FramePayload) Metadata() ([]byte, bool) {
	return p.trySliceMetadata(0)
}

// Data returns data bytes.
func (p *FramePayload) Data() []byte {
	return p.trySliceData(0)
}

// MetadataUTF8 returns metadata as UTF8 string.
func (p *FramePayload) MetadataUTF8() (metadata string, ok bool) {
	raw, ok := p.Metadata()
	if ok {
		metadata = common.Bytes2str(raw)
	}
	return
}

// DataUTF8 returns data as UTF8 string.
func (p *FramePayload) DataUTF8() string {
	return common.Bytes2str(p.Data())
}

// NewFramePayload returns a new payload frame.
func NewFramePayload(id uint32, data, metadata []byte, flags ...FrameFlag) *FramePayload {
	fg := newFlags(flags...)
	bf := common.New()
	if len(metadata) > 0 {
		fg |= FlagMetadata
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
	return &FramePayload{
		NewBaseFrame(NewFrameHeader(id, FrameTypePayload, fg), bf),
	}
}
