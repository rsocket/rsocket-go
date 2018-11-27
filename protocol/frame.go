package protocol

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
	REQUEST_IN       FrameType = 0x08
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
	case REQUEST_IN:
		return "REQUEST_IN"
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
	f0 Flags = 1 << iota
	f1
	f2
	f3
	f4
	f5
	f6
	f7
	f8
	f9
)

func (f Flags) check(mask Flags) bool {
	return mask&f == mask
}

type Frame []byte

func (p Frame) IsIgnore() bool {
	return p.Flags().check(f9)
}

func (p Frame) IsMetadata() bool {
	return p.Flags().check(f8)
}

func (p Frame) StreamID() uint32 {
	return byteOrder.Uint32(p[:4])
}

func (p Frame) Type() FrameType {
	foo := byteOrder.Uint16(p[4:6])
	return FrameType((foo & 0xFC00) >> 10)
}

func (p Frame) Flags() Flags {
	return Flags(byteOrder.Uint16(p[4:6]))
}

func (p Frame) sliceMetadata(offset int) []byte {
	if !p.IsMetadata() {
		return nil
	}
	l := readUint24(p, offset)
	offset += 3
	return p[offset : offset+l]
}

func (p Frame) slicePayload(offset int) []byte {
	if p.IsMetadata() {
		offset += 3 + readUint24(p, offset)
	}
	return p[offset:]
}

type FrameHandler = func(frame Frame) error

type FrameDecoder interface {
	Handle(fn FrameHandler) error
}
