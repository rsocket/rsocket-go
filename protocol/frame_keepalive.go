package protocol

type FrameKeepalive struct {
	Frame
}

func (p FrameKeepalive) IsRespond() bool {
	return p.Frame.Flags().check(FlagRespond)
}

func (p FrameKeepalive) LastReceivedPosition() uint64 {
	return byteOrder.Uint64(p.Frame[frameHeaderLength : frameHeaderLength+8])
}

func (p FrameKeepalive) Payload() []byte {
	return p.slicePayload(frameHeaderLength + 8)
}
