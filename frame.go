package rsocket

import (
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


func sliceMetadataAndData(header *Header, raw []byte, offset int) (metadata []byte, data []byte) {
	if !header.Flags().Check(FlagMetadata) {
		foo := raw[offset:]
		data = make([]byte, len(foo))
		copy(data, foo)
		return
	}
	l := decodeU24(raw, offset)
	offset += 3
	metadata = make([]byte, l)
	copy(metadata, raw[offset:offset+l])
	foo := raw[offset+l:]
	data = make([]byte, len(foo))
	copy(data, foo)
	return
}

type Frame interface {
	io.WriterTo
	Size() int
}
