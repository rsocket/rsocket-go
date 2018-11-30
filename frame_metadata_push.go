package rsocket

type FrameMetadataPush struct {
	*Header
	metadata []byte
}

func (p *FrameMetadataPush) Metadata() []byte {
	return p.metadata
	//return p.Frame[frameHeaderLength:]
}

func asMetadataPush(h *Header, raw []byte) *FrameMetadataPush {
	return &FrameMetadataPush{
		Header:   h,
		metadata: sliceMetadata(h, raw, frameHeaderLength),
	}
}

func mkMetadataPush(sid uint32, metadata []byte, f ...Flags) *FrameMetadataPush {
	return &FrameMetadataPush{
		Header:   mkHeader(sid, METADATA_PUSH, f...),
		metadata: metadata,
	}
}
