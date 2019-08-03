package framing

import (
	"errors"
	"io"
	"strings"

	"github.com/rsocket/rsocket-go/internal/common"
)

var errIncompleteFrame = errors.New("incomplete frame")

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
func (f FrameFlag) Check(mask FrameFlag) bool {
	return mask&f == mask
}

func newFlags(flags ...FrameFlag) FrameFlag {
	var fg FrameFlag
	for _, it := range flags {
		fg |= it
	}
	return fg
}

// Frame is a single message containing a request, response, or protocol processing.
type Frame interface {
	io.WriterTo
	// Header returns frame FrameHeader.
	Header() FrameHeader
	Body() *common.ByteBuff
	// Len returns length of frame.
	Len() int
	// Validate returns error if frame is invalid.
	Validate() error
	// SetHeader set frame header.
	SetHeader(h FrameHeader)
	// SetBody set frame body.
	SetBody(body *common.ByteBuff)
	// Bytes encodes and returns frame in bytes.
	Bytes() []byte
	// IsResumable returns true if frame supports resume.
	IsResumable() bool
}

// BaseFrame is basic frame implementation.
type BaseFrame struct {
	header   FrameHeader
	body     *common.ByteBuff
}

// IsResumable returns true if frame supports resume.
func (p *BaseFrame) IsResumable() bool {
	switch p.header.Type() {
	case FrameTypeRequestChannel, FrameTypeRequestStream, FrameTypeRequestResponse, FrameTypeRequestFNF, FrameTypeRequestN, FrameTypeCancel, FrameTypeError, FrameTypePayload:
		return true
	default:
		return false
	}
}

// SetBody set frame body.
func (p *BaseFrame) SetBody(body *common.ByteBuff) {
	p.body = body
}

// Body returns frame body.
func (p *BaseFrame) Body() *common.ByteBuff {
	return p.body
}

// Header returns frame header.
func (p *BaseFrame) Header() FrameHeader {
	return p.header
}

// SetHeader set frame header.
func (p *BaseFrame) SetHeader(h FrameHeader) {
	p.header = h
}

// Len returns length of frame.
func (p *BaseFrame) Len() int {
	return HeaderLen + p.body.Len()
}

// WriteTo write frame to writer.
func (p *BaseFrame) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int64
	wrote, err = p.header.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote
	wrote, err = p.body.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote
	return
}

// Bytes returns frame in bytes.
func (p *BaseFrame) Bytes() []byte {
	return append(p.header[:], p.body.Bytes()...)
}


// NewBaseFrame returns a new BaseFrame.
func NewBaseFrame(h FrameHeader, body *common.ByteBuff) (f *BaseFrame) {
	f = &BaseFrame{
		header:   h,
		body:     body,
	}
	return
}

func (p *BaseFrame) trySeekMetadataLen(offset int) (n int, hasMetadata bool) {
	raw := p.body.Bytes()
	if offset > 0 {
		raw = raw[offset:]
	}
	hasMetadata = p.header.Flag().Check(FlagMetadata)
	if !hasMetadata {
		return
	}
	if len(raw) < 3 {
		n = -1
	} else {
		n = common.NewUint24Bytes(raw).AsInt()
	}
	return
}

func (p *BaseFrame) trySliceMetadata(offset int) ([]byte, bool) {
	n, ok := p.trySeekMetadataLen(offset)
	if !ok || n < 0 {
		return nil, false
	}
	return p.body.Bytes()[offset+3 : offset+3+n], true
}

func (p *BaseFrame) trySliceData(offset int) []byte {
	n, ok := p.trySeekMetadataLen(offset)
	if !ok {
		return p.body.Bytes()[offset:]
	}
	if n < 0 {
		return nil
	}
	return p.body.Bytes()[offset+n+3:]
}
