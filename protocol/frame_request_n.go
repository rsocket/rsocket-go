package protocol

type FrameRequestN struct {
	Frame
}

func (p FrameRequestN) RequestN() uint32 {
	return byteOrder.Uint32(p.Frame[frameHeaderLength : frameHeaderLength+4])
}
