package rsocket

import "encoding/binary"

type FrameRequestN struct {
	*Header
	n uint32
}

func (p *FrameRequestN) RequestN() uint32 {
	return p.n
}

func asRequestN(h *Header, raw []byte) *FrameRequestN {
	n := binary.BigEndian.Uint32(raw[frameHeaderLength : frameHeaderLength+4])
	return &FrameRequestN{
		Header: h,
		n:      n,
	}
}

func mkRequestN(sid uint32, n uint32, f ...Flags) *FrameRequestN {
	h := mkHeader(sid, REQUEST_N, f...)
	return &FrameRequestN{
		Header: h,
		n:      n,
	}
}
