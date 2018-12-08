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
	m, d := sliceMetadataAndData(h, raw, headerLen+4)
	return &FrameRequestStream{
		Header:          h,
		initialRequestN: binary.BigEndian.Uint32(raw[headerLen : headerLen+4]),
		metadata:        m,
		data:            d,
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
