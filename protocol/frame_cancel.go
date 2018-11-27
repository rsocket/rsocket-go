package protocol

type FrameCancel struct {
	Frame
}

func (p FrameCancel) IsFollow() bool {
	return p.Frame.Flags().check(f7)
}

func (p FrameCancel) IsComplete() bool {
	return p.Frame.Flags().check(f6)
}

func (p FrameCancel) IsNext() bool {
	return p.Frame.Flags().check(f5)
}

func (p FrameCancel) Metadata() []byte {
	return p.Frame.sliceMetadata(frameHeaderLength)
}

func (p FrameCancel) Payload() []byte {
	return p.Frame.slicePayload(frameHeaderLength)
}
