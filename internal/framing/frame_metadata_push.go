package framing

import (
	"fmt"

	"github.com/rsocket/rsocket-go/internal/common"
)

var defaultFrameMetadataPushHeader = NewFrameHeader(0, FrameTypeMetadataPush, FlagMetadata)

// FrameMetadataPush is metadata push frame.
type FrameMetadataPush struct {
	*BaseFrame
}

// Validate returns error if frame is invalid.
func (p *FrameMetadataPush) Validate() (err error) {
	return
}

func (p *FrameMetadataPush) String() string {
	m, _ := p.MetadataUTF8()
	return fmt.Sprintf("FrameMetadataPush{%s,metadata=%s}", p.header, m)
}

// Metadata returns metadata bytes.
func (p *FrameMetadataPush) Metadata() ([]byte, bool) {
	return p.body.Bytes(), true
}

// Data returns data bytes.
func (p *FrameMetadataPush) Data() []byte {
	return nil
}

// MetadataUTF8 returns metadata as UTF8 string.
func (p *FrameMetadataPush) MetadataUTF8() (metadata string, ok bool) {
	raw, ok := p.Metadata()
	if ok {
		metadata = string(raw)
	}
	return
}

// DataUTF8 returns data as UTF8 string.
func (p *FrameMetadataPush) DataUTF8() (data string) {
	return
}

// NewFrameMetadataPush returns a new metadata push frame.
func NewFrameMetadataPush(metadata []byte) *FrameMetadataPush {
	bf := common.New()
	if _, err := bf.Write(metadata); err != nil {
		panic(err)
	}
	return &FrameMetadataPush{
		NewBaseFrame(defaultFrameMetadataPushHeader, bf),
	}
}
