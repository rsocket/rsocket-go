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
	m, d := sliceMetadataAndData(h, raw, headerLen)
	return &FrameFNF{
		Header:   h,
		metadata: m,
		data:     d,
	}
}
