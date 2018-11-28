package protocol

type FrameRequestChannel struct {
	Frame
}

func (p FrameRequestChannel) IsFollow() bool {
	return p.Frame.Flags().check(FlagFollow)
}

func (p FrameRequestChannel) IsComplete() bool {
	return p.Frame.Flags().check(FlagComplete)
}

func (p FrameRequestChannel) InitialRequestN() uint32 {
	return byteOrder.Uint32(p.Frame[frameHeaderLength : frameHeaderLength+4])
}

func (p FrameRequestChannel) Metadata() []byte {
	return p.Frame.sliceMetadata(frameHeaderLength + 4)
}

func (p FrameRequestChannel) Payload() []byte {
	return p.Frame.slicePayload(frameHeaderLength + 4)
}
