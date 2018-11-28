package protocol

type FrameRequestResponse struct {
	Frame
}

func (p FrameRequestResponse) IsFollow() bool {
	return p.Frame.Flags().check(FlagFollow)
}

func (p FrameRequestResponse) Metadata() []byte {
	return p.Frame.sliceMetadata(frameHeaderLength)
}

func (p FrameRequestResponse) Payload() []byte {
	return p.Frame.slicePayload(frameHeaderLength)
}
