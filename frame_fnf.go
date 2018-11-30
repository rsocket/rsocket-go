package rsocket

type FrameFNF struct {
	*Header
	metadata []byte
	data     []byte
}

func (p *FrameFNF) Metadata() []byte {
	return p.metadata
}

func (p *FrameFNF) Data() []byte {
	return p.data
}

func asFNF(h *Header, raw []byte) *FrameFNF {
	return &FrameFNF{
		Header:   h,
		metadata: sliceMetadata(h, raw, frameHeaderLength),
		data:     sliceData(h, raw, frameHeaderLength),
	}
}
