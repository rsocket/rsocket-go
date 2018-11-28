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

func NewFrameKeepalive(base *BaseFrame, responsd bool, position uint64, data []byte) Frame {
	bs := make([]byte, 0)
	if responsd {
		base.SetFlag(FlagRespond)
	}
	bs = append(bs, base.asBytes()...)
	b := make([]byte, 8)
	byteOrder.PutUint64(b, position)
	bs = append(bs, b...)
	if data != nil && len(data) > 0 {
		bs = append(bs, data...)
	}
	return Frame(bs)
}
