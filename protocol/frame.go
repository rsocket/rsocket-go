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

var (
	frameTypeAlias = map[FrameType]string{
		RESERVED:         "RESERVED",
		SETUP:            "SETUP",
		LEASE:            "LEASE",
		KEEPALIVE:        "KEEPALIVE",
		REQUEST_RESPONSE: "REQUEST_RESPONSE",
		REQUEST_FNF:      "REQUEST_FNF",
		REQUEST_STREAM:   "REQUEST_STREAM",
		REQUEST_CHANNEL:  "REQUEST_CHANNEL",
		REQUEST_IN:       "REQUEST_IN",
		CANCEL:           "CANCEL",
		PAYLOAD:          "PAYLOAD",
		ERROR:            "ERROR",
		METADATA_PUSH:    "METADATA_PUSH",
		RESUME:           "RESUME",
		RESUME_OK:        "RESUME_OK",
		EXT:              "EXT",
	}
)

type FramePayload []byte

type FrameHeader struct {
	StreamID   uint32
	Type       FrameType
	IsIgnore   bool
	IsMetadata bool
	Flags      uint8
}

type Frame struct {
	Header  FrameHeader
	Payload *FramePayload
}

type FrameHandler = func(frame *Frame) error

type FrameDecoder interface {
	Handle(fn FrameHandler) error
}
