package protocol

type FramePayload struct {
	Frame
}

func (p FramePayload) IsFollow() bool {
	return p.Frame.Flags().check(FlagFollow)
}

func (p FramePayload) IsComplete() bool {
	return p.Frame.Flags().check(FlagComplete)
}

func (p FramePayload) IsNext() bool {
	return p.Frame.Flags().check(FlagNext)
}

func (p FramePayload) Metadata() []byte {
	return p.Frame.sliceMetadata(frameHeaderLength)
}

func (p FramePayload) Payload() []byte {
	return p.Frame.slicePayload(frameHeaderLength)
}

func NewPayload(base *BaseFrame, payload []byte, metadata []byte) Frame {
	bs := make([]byte, 0)
	bs = append(bs, base.asBytes()...)
	if base.Flags.check(FlagMetadata) {
		bs = append(bs, writeUint24(len(metadata))...)
		bs = append(bs, metadata...)
	}
	bs = append(bs, payload...)
	return Frame(bs)
}
