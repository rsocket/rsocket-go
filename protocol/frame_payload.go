package protocol

type FramePayload struct {
	Frame
}

func (p FramePayload) IsFollow() bool {
	return p.Frame.Flags().check(f7)
}

func (p FramePayload) IsComplete() bool {
	return p.Frame.Flags().check(f6)
}

func (p FramePayload) IsNext() bool {
	return p.Frame.Flags().check(f5)
}

func (p FramePayload) Metadata() []byte {
	return p.Frame.sliceMetadata(frameHeaderLength)
}

func (p FramePayload) Payload() []byte {
	return p.Frame.slicePayload(frameHeaderLength)
}
