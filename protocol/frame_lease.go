package protocol

import "time"

type FrameLease struct {
	Frame
}

func (p FrameLease) TimeToLive() time.Duration {
	mills := byteOrder.Uint32(p.Frame[frameHeaderLength : frameHeaderLength+4])
	return time.Millisecond * time.Duration(mills)
}

func (p FrameLease) NumberOfRequests() uint32 {
	return byteOrder.Uint32(p.Frame[frameHeaderLength+4 : frameHeaderLength+8])
}

func (p FrameLease) Metadata() []byte {
	return p.Frame.sliceMetadata(frameHeaderLength + 8)
}
