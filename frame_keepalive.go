package rsocket

import (
	"encoding/binary"
)

type FrameKeepalive struct {
	*Header
	lastReceivedPosition uint64
	data                 []byte
}

func (p *FrameKeepalive) Bytes() []byte {
	bs := p.Header.Bytes()
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, p.lastReceivedPosition)
	bs = append(bs, b...)
	if p.data != nil {
		bs = append(bs, p.data...)
	}
	return bs
}

func (p *FrameKeepalive) LastReceivedPosition() uint64 {
	return p.lastReceivedPosition
}

func (p *FrameKeepalive) Data() []byte {
	return p.data
}

func asKeepalive(h *Header, raw []byte) *FrameKeepalive {
	pos := binary.BigEndian.Uint64(raw[headerLen : headerLen+8])
	data := raw[headerLen+8:]
	return &FrameKeepalive{
		Header:               h,
		lastReceivedPosition: pos,
		data:                 data,
	}
}

func mkKeepalive(sid uint32, pos uint64, data []byte, f ...Flags) *FrameKeepalive {
	h := mkHeader(sid, KEEPALIVE, f...)
	return &FrameKeepalive{
		Header:               h,
		lastReceivedPosition: pos,
		data:                 data,
	}
}
