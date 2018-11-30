package rsocket

import (
	"fmt"
)

type FrameRequestResponse struct {
	*Header
	metadata []byte
	data     []byte
}

func (p *FrameRequestResponse) String() string {
	return fmt.Sprintf("FrameRequestResponse{Headeer=%+v, metadata=%s, data=%s}", p.Header, string(p.metadata), string(p.data))
}

func (p *FrameRequestResponse) Metadata() []byte {
	return p.metadata
}

func (p *FrameRequestResponse) Data() []byte {
	return p.data
}

func (p *FrameRequestResponse) Bytes() []byte {
	bs := p.Header.Bytes()
	if p.Header.Flags().Check(FlagMetadata) {
		bs = append(bs, encodeU24(len(p.metadata))...)
		bs = append(bs, p.metadata...)
	}
	bs = append(bs, p.data...)
	return bs
}

func asRequestResponse(h *Header, raw []byte) *FrameRequestResponse {
	return &FrameRequestResponse{
		Header:   h,
		metadata: sliceMetadata(h, raw, frameHeaderLength),
		data:     sliceData(h, raw, frameHeaderLength),
	}
}

func mkRequestResponse(sid uint32, metadata []byte, data []byte, f ...Flags) *FrameRequestResponse {
	h := mkHeader(sid, REQUEST_RESPONSE, f...)
	return &FrameRequestResponse{
		Header:   h,
		metadata: metadata,
		data:     data,
	}
}
