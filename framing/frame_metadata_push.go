package framing

import (
	"fmt"
	"github.com/rsocket/rsocket-go/common"
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
	return fmt.Sprintf("FrameMetadataPush{%s,metadata=%s}", p.header, p.Metadata())
}

// Metadata returns metadata bytes.
func (p *FrameMetadataPush) Metadata() []byte {
	return p.body.Bytes()
}

// Data returns data bytes.
func (p *FrameMetadataPush) Data() []byte {
	return nil
}

// NewFrameMetadataPush returns a new metadata push frame.
func NewFrameMetadataPush(metadata []byte) *FrameMetadataPush {
	bf := common.BorrowByteBuffer()
	_, _ = bf.Write(metadata)
	return &FrameMetadataPush{
		&BaseFrame{
			header: defaultFrameMetadataPushHeader,
			body:   bf,
		},
	}
}
