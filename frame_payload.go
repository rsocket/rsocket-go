package rsocket

type FramePayload struct {
	*Header
	metadata []byte
	data     []byte
}

func (p *FramePayload) Metadata() []byte {
	return p.metadata
}

func (p *FramePayload) Data() []byte {
	return p.data
}

func (p *FramePayload) Bytes() []byte {
	bs := p.Header.Bytes()
	if p.Header.Flags().Check(FlagMetadata) {
		bs = append(bs, encodeU24(len(p.metadata))...)
		bs = append(bs, p.metadata...)
	}
	if p.data != nil {
		bs = append(bs, p.data...)
	}
	return bs
}

func asPayload(h *Header, raw []byte) *FramePayload {
	return &FramePayload{
		Header:   h,
		metadata: sliceMetadata(h, raw, frameHeaderLength),
		data:     sliceData(h, raw, frameHeaderLength),
	}
}

func mkPayload(sid uint32, metadata []byte, data []byte, f ...Flags) *FramePayload {
	return &FramePayload{
		Header:   mkHeader(sid, PAYLOAD, f...),
		metadata: metadata,
		data:     data,
	}
}
