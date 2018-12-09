package rsocket

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
)

type FrameType uint8

const (
	RESERVED         FrameType = 0x00
	SETUP            FrameType = 0x01
	LEASE            FrameType = 0x02
	KEEPALIVE        FrameType = 0x03
	REQUEST_RESPONSE FrameType = 0x04
	REQUEST_FNF      FrameType = 0x05
	REQUEST_STREAM   FrameType = 0x06
	REQUEST_CHANNEL  FrameType = 0x07
	REQUEST_N        FrameType = 0x08
	CANCEL           FrameType = 0x09
	PAYLOAD          FrameType = 0x0A
	ERROR            FrameType = 0x0B
	METADATA_PUSH    FrameType = 0x0C
	RESUME           FrameType = 0x0D
	RESUME_OK        FrameType = 0x0E
	EXT              FrameType = 0x3F
)

func (f FrameType) String() string {
	switch f {
	case RESERVED:
		return "RESERVED"
	case SETUP:
		return "SETUP"
	case LEASE:
		return "LEASE"
	case KEEPALIVE:
		return "KEEPALIVE"
	case REQUEST_RESPONSE:
		return "REQUEST_RESPONSE"
	case REQUEST_FNF:
		return "REQUEST_FNF"
	case REQUEST_STREAM:
		return "REQUEST_STREAM"
	case REQUEST_CHANNEL:
		return "REQUEST_CHANNEL"
	case REQUEST_N:
		return "REQUEST_N"
	case CANCEL:
		return "CANCEL"
	case PAYLOAD:
		return "PAYLOAD"
	case ERROR:
		return "ERROR"
	case METADATA_PUSH:
		return "METADATA_PUSH"
	case RESUME:
		return "RESUME"
	case RESUME_OK:
		return "RESUME_OK"
	case EXT:
		return "EXT"
	default:
		return "UNKNOWN"
	}
}

type Flags uint16

const (
	FlagNext Flags = 1 << (5 + iota)
	FlagComplete
	FlagFollow
	FlagMetadata
	FlagIgnore

	FlagResume  = FlagFollow
	FlagLease   = FlagComplete
	FlagRespond = FlagFollow
)

func (f Flags) Check(mask Flags) bool {
	return mask&f == mask
}

type Frame interface {
	io.WriterTo
	Size() int
}

type FrameHandler = func(h *Header, raw []byte) error

type FrameDecoder interface {
	Handle(ctx context.Context, fn FrameHandler) error
}

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

func asHeader(bs []byte) (*Header, error) {
	id := binary.BigEndian.Uint32(bs[:4])
	n := binary.BigEndian.Uint16(bs[4:6])
	return &Header{
		streamID:  id,
		frameType: FrameType((n & 0xFC00) >> 10),
		flags:     Flags(n & 0x03FF),
	}, nil
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
