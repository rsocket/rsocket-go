package protocol

type FrameRequestStream struct {
	Frame
}

func (p FrameRequestStream) IsFollow() bool {
	return p.Frame.Flags().check(FlagFollow)

}

func (p FrameRequestStream) InitialRequestN() uint32 {
	return byteOrder.Uint32(p.Frame[frameHeaderLength : frameHeaderLength+4])
}

func (p FrameRequestStream) Metadata() []byte {
	return p.sliceMetadata(frameHeaderLength + 4)
}

func (p FrameRequestStream) Payload() []byte {
	return p.slicePayload(frameHeaderLength + 4)
}
