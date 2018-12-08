package rsocket

import "encoding/binary"

type FrameRequestChannel struct {
	*Header
	initialRequestN uint32
	meatadata       []byte
	data            []byte
}

func (p *FrameRequestChannel) InitialRequestN() uint32 {
	return p.initialRequestN
}

func (p *FrameRequestChannel) Metadata() []byte {
	return p.meatadata
}

func (p *FrameRequestChannel) Payload() []byte {
	return p.data
}

func asRequestChannel(h *Header, raw []byte) *FrameRequestChannel {
	m, d := sliceMetadataAndData(h, raw, headerLen+4)
	return &FrameRequestChannel{
		Header:          h,
		initialRequestN: binary.BigEndian.Uint32(raw[headerLen : headerLen+4]),
		meatadata:       m,
		data:            d,
	}
}

func mkRequestChannel(sid uint32, initalRequestN uint32, metadata []byte, data []byte, f ...Flags) *FrameRequestChannel {
	h := mkHeader(sid, REQUEST_CHANNEL, f...)
	return &FrameRequestChannel{
		Header:          h,
		initialRequestN: initalRequestN,
		meatadata:       metadata,
		data:            data,
	}
}
