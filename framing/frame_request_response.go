package framing

import (
	"fmt"
	"github.com/rsocket/rsocket-go/common"
)

// FrameRequestResponse is frame for requesting single response.
type FrameRequestResponse struct {
	*BaseFrame
}

// Validate returns error if frame is invalid.
func (p *FrameRequestResponse) Validate() (err error) {
	return
}

func (p *FrameRequestResponse) String() string {
	return fmt.Sprintf("FrameRequestResponse{%s,data=%s,metadata=%s}", p.header, p.Data(), p.Metadata())
}

// Metadata returns metadata bytes.
func (p *FrameRequestResponse) Metadata() []byte {
	return p.trySliceMetadata(0)
}

// Data returns data bytes.
func (p *FrameRequestResponse) Data() []byte {
	return p.trySliceData(0)
}

// NewFrameRequestResponse returns a new RequestResponse frame.
func NewFrameRequestResponse(id uint32, data, metadata []byte, flags ...FrameFlag) *FrameRequestResponse {
	fg := newFlags(flags...)
	bf := common.BorrowByteBuffer()
	if len(metadata) > 0 {
		fg |= FlagMetadata
		_ = bf.WriteUint24(len(metadata))
		_, _ = bf.Write(metadata)
	}
	if len(data) > 0 {
		_, _ = bf.Write(data)
	}
	return &FrameRequestResponse{
		&BaseFrame{
			header: NewFrameHeader(id, FrameTypeRequestResponse, fg),
			body:   bf,
		},
	}
}
