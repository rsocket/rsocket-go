package core

import (
	"io"
	"strings"

	"github.com/rsocket/rsocket-go/internal/common"
)

// FrameType is type of frame.
type FrameType uint8

// All frame types
const (
	FrameTypeReserved        FrameType = 0x00
	FrameTypeSetup           FrameType = 0x01
	FrameTypeLease           FrameType = 0x02
	FrameTypeKeepalive       FrameType = 0x03
	FrameTypeRequestResponse FrameType = 0x04
	FrameTypeRequestFNF      FrameType = 0x05
	FrameTypeRequestStream   FrameType = 0x06
	FrameTypeRequestChannel  FrameType = 0x07
	FrameTypeRequestN        FrameType = 0x08
	FrameTypeCancel          FrameType = 0x09
	FrameTypePayload         FrameType = 0x0A
	FrameTypeError           FrameType = 0x0B
	FrameTypeMetadataPush    FrameType = 0x0C
	FrameTypeResume          FrameType = 0x0D
	FrameTypeResumeOK        FrameType = 0x0E
	FrameTypeExt             FrameType = 0x3F
)

func (f FrameType) String() string {
	switch f {
	case FrameTypeReserved:
		return "RESERVED"
	case FrameTypeSetup:
		return "SETUP"
	case FrameTypeLease:
		return "LEASE"
	case FrameTypeKeepalive:
		return "KEEPALIVE"
	case FrameTypeRequestResponse:
		return "REQUEST_RESPONSE"
	case FrameTypeRequestFNF:
		return "REQUEST_FNF"
	case FrameTypeRequestStream:
		return "REQUEST_STREAM"
	case FrameTypeRequestChannel:
		return "REQUEST_CHANNEL"
	case FrameTypeRequestN:
		return "REQUEST_N"
	case FrameTypeCancel:
		return "CANCEL"
	case FrameTypePayload:
		return "PAYLOAD"
	case FrameTypeError:
		return "ERROR"
	case FrameTypeMetadataPush:
		return "METADATA_PUSH"
	case FrameTypeResume:
		return "RESUME"
	case FrameTypeResumeOK:
		return "RESUME_OK"
	case FrameTypeExt:
		return "EXT"
	default:
		return "UNKNOWN"
	}
}

// FrameFlag is flag of frame.
type FrameFlag uint16

func (f FrameFlag) String() string {
	foo := make([]string, 0)
	if f.Check(FlagNext) {
		foo = append(foo, "N")
	}
	if f.Check(FlagComplete) {
		foo = append(foo, "CL")
	}
	if f.Check(FlagFollow) {
		foo = append(foo, "FRS")
	}
	if f.Check(FlagMetadata) {
		foo = append(foo, "M")
	}
	if f.Check(FlagIgnore) {
		foo = append(foo, "I")
	}
	return strings.Join(foo, "|")
}

// All frame flags
const (
	FlagNext FrameFlag = 1 << (5 + iota)
	FlagComplete
	FlagFollow
	FlagMetadata
	FlagIgnore

	FlagResume  = FlagFollow
	FlagLease   = FlagComplete
	FlagRespond = FlagFollow
)

// Check returns true if mask exists.
func (f FrameFlag) Check(flag FrameFlag) bool {
	return flag&f == flag
}

type Frame interface {
	// FrameHeader returns frame FrameHeader.
	Header() FrameHeader
	// Len returns length of frame.
	Len() int
}

// WriteableFrame means writeable frame.
type WriteableFrame interface {
	Frame
	io.WriterTo
	// Done marks current frame has been sent.
	Done()
	HandleDone(func())
}

// BufferedFrame is a single message containing a request, response, or protocol processing.
type BufferedFrame interface {
	Frame
	io.WriterTo
	common.Releasable
	// Validate returns error if frame is invalid.
	Validate() error
	// HasFlag returns true if target frame flag is enabled.
	HasFlag(FrameFlag) bool
	// StreamID returns the stream id of current frame.
	StreamID() uint32
}
