package rsocket

import "io"

type FrameMetadataPush struct {
	*Header
	metadata []byte
}

func (p *FrameMetadataPush) WriteTo(w io.Writer) (n int64, err error) {
	panic("implement me")
}

func (p *FrameMetadataPush) Size() int {
	panic("implement me")
}

func (p *FrameMetadataPush) Metadata() []byte {
	return p.metadata
}

func asMetadataPush(h *Header, raw []byte) *FrameMetadataPush {
	return &FrameMetadataPush{
		Header:   h,
		metadata: sliceMetadata(h, raw, headerLen),
	}
}

func mkMetadataPush(sid uint32, metadata []byte, f ...Flags) *FrameMetadataPush {
	return &FrameMetadataPush{
		Header:   mkHeader(sid, METADATA_PUSH, f...),
		metadata: metadata,
	}
}
