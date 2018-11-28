package protocol

type FrameMetadataPush struct {
	Frame
}

func (p FrameMetadataPush) Metadata() []byte {
	return p.Frame[frameHeaderLength:]
}
