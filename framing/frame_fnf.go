package framing

import (
	"fmt"
	"github.com/rsocket/rsocket-go/common"
)

// FrameFNF is fire and forget frame.
type FrameFNF struct {
	*BaseFrame
}

// Validate returns error if frame is invalid.
func (p *FrameFNF) Validate() (err error) {
	return
}

func (p *FrameFNF) String() string {
	return fmt.Sprintf("FrameFNF{%s,data=%s,metadata=%s}", p.header, p.Data(), p.Metadata())
}

// Metadata returns metadata bytes.
func (p *FrameFNF) Metadata() []byte {
	return p.trySliceMetadata(0)
}

// Data returns data bytes.
func (p *FrameFNF) Data() []byte {
	return p.trySliceData(0)
}

// NewFrameFNF returns a new fire and forget frame.
func NewFrameFNF(sid uint32, data, metadata []byte, flags ...FrameFlag) *FrameFNF {
	fg := newFlags(flags...)
	bf := common.BorrowByteBuffer()
	if len(metadata) > 0 {
		fg |= FlagMetadata
		_ = bf.WriteUint24(len(metadata))
		_, _ = bf.Write(metadata)
	}
	_, _ = bf.Write(data)
	return &FrameFNF{
		&BaseFrame{
			header: NewFrameHeader(sid, FrameTypeRequestFNF, fg),
			body:   bf,
		},
	}
}
