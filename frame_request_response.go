package rsocket

import "fmt"

type frameRequestResponse struct {
	*baseFrame
}

func (p *frameRequestResponse) String() string {
	return fmt.Sprintf("frameRequestResponse{%s,data=%s,metadata=%s}", p.header, p.Data(), p.Metadata())
}

func (p *frameRequestResponse) Metadata() []byte {
	return p.trySliceMetadata(0)
}
func (p *frameRequestResponse) Data() []byte {
	return p.trySliceData(0)
}

func createRequestResponse(id uint32, data, metadata []byte, flags ...rFlags) *frameRequestResponse {
	fg := newFlags(flags...)
	bf := borrowByteBuffer()
	if len(metadata) > 0 {
		fg |= flagMetadata
		_ = bf.WriteUint24(len(metadata))
		_, _ = bf.Write(metadata)
	}
	if len(data) > 0 {
		_, _ = bf.Write(data)
	}
	return &frameRequestResponse{
		&baseFrame{
			header: createHeader(id, tRequestResponse, fg),
			body:   bf,
		},
	}
}
