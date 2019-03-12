package rsocket

import (
	"fmt"
)

type framePayload struct {
	*baseFrame
}

func (p *framePayload) validate() (err error) {
	return
}

func (p *framePayload) String() string {
	return fmt.Sprintf("framePayload{%s,data=%s,metadata=%s}", p.header, string(p.Data()), string(p.Metadata()))
}

func (p *framePayload) Metadata() []byte {
	return p.trySliceMetadata(0)
}

func (p *framePayload) Data() []byte {
	return p.trySliceData(0)
}

func createPayloadFrame(id uint32, data, metadata []byte, flags ...rFlags) *framePayload {
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
	return &framePayload{
		&baseFrame{
			header: createHeader(id, tPayload, fg),
			body:   bf,
		},
	}
}
