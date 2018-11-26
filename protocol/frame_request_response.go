package protocol

type FrameRequestResponse struct {
	Frame
}

func (p FrameRequestResponse) IsFollow() bool {
	return p.Frame.GetFlags().check(f7)
}

func (p FrameRequestResponse) GetMetadataPayload() []byte {
	if !p.IsMetadata() {
		return nil
	}
	l := readUint24WithOffset(p.Frame, 6)
	return p.Frame[9 : 9+l]
}

func (p FrameRequestResponse) GetPayload() []byte {
	if !p.IsMetadata() {
		return p.Frame[6:]
	}
	l := readUint24WithOffset(p.Frame, 6)
	return p.Frame[9+l:]
}
