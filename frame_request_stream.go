package rsocket

import "encoding/binary"

type FrameRequestStream struct {
	*Header
	initialRequestN uint32
	metadata        []byte
	data            []byte
}

func (p FrameRequestStream) InitialRequestN() uint32 {
	return p.initialRequestN
}

func (p FrameRequestStream) Metadata() []byte {
	return p.metadata
}

func (p FrameRequestStream) Data() []byte {
	return p.data
}

func asRequestStream(h *Header, raw []byte) *FrameRequestStream {
	return &FrameRequestStream{
		Header:          h,
		initialRequestN: binary.BigEndian.Uint32(raw[frameHeaderLength : frameHeaderLength+4]),
		metadata:        sliceMetadata(h, raw, frameHeaderLength+4),
		data:            sliceData(h, raw, frameHeaderLength+4),
	}
}

func mkRequestStream(sid uint32, n uint32, metadata []byte, data []byte, f ...Flags) *FrameRequestStream {
	return &FrameRequestStream{
		Header:          mkHeader(sid, REQUEST_STREAM, f...),
		initialRequestN: n,
		metadata:        metadata,
		data:            data,
	}
}
