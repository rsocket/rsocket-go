package protocol

type FrameFNF struct {
	Frame
}

func (p FrameFNF) IsFollow() bool {
	return p.Frame.Flags().check(FlagFollow)
}

func (p FrameFNF) Metadata() []byte {
	return p.Frame.sliceMetadata(frameHeaderLength)
}

func (p FrameFNF) Payload() []byte {
	return p.Frame.slicePayload(frameHeaderLength)
}
