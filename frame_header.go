package rsocket

import (
	"encoding/binary"
	"fmt"
)

type Header struct {
	streamID  uint32
	frameType FrameType
	flags     Flags
}

func (p *Header) String() string {
	return fmt.Sprintf("Header{StreamID=%d, Type=%s, Flags=%X}", p.streamID, p.frameType, p.flags)
}

func (p *Header) StreamID() uint32 {
	return p.streamID
}

func (p *Header) Type() FrameType {
	return p.frameType
}

func (p *Header) Flags() Flags {
	return p.flags
}

func (p *Header) Bytes() []byte {
	bs := make([]byte, headerLen)
	binary.BigEndian.PutUint32(bs, p.streamID)
	binary.BigEndian.PutUint16(bs[4:], uint16(p.frameType)<<10|uint16(p.flags))
	return bs
}

func (p *Header) Parse(bs []byte) error {
	p.streamID = binary.BigEndian.Uint32(bs[:4])
	n := binary.BigEndian.Uint16(bs[4:6])
	p.frameType = FrameType((n & 0xFC00) >> 10)
	p.flags = Flags(n & 0x03FF)
	return nil
}

func mkHeader(sid uint32, t FrameType, f ...Flags) *Header {
	var fg uint16
	for _, it := range f {
		fg |= uint16(it)
	}
	return &Header{
		streamID:  sid,
		flags:     Flags(fg),
		frameType: t,
	}
}
