package protocol

type FrameExtension struct {
	Frame
}

func (p FrameExtension) ExtendedType() uint32 {
	return byteOrder.Uint32(p.Frame[frameHeaderLength : frameHeaderLength+4])
}
