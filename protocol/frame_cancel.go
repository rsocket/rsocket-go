package protocol

type FrameCancel struct {
	Frame
}

func (p FrameCancel) IsFollow() bool {
	return p.Frame.Flags().check(FlagFollow)
}

func (p FrameCancel) IsComplete() bool {
	return p.Frame.Flags().check(FlagComplete)
}

func (p FrameCancel) IsNext() bool {
	return p.Frame.Flags().check(FlagNext)
}

func (p FrameCancel) Metadata() []byte {
	return p.Frame.sliceMetadata(frameHeaderLength)
}

func (p FrameCancel) Payload() []byte {
	return p.Frame.slicePayload(frameHeaderLength)
}
