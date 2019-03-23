package framing

import (
	"errors"
	"github.com/rsocket/rsocket-go/common"
	"io"
	"strings"
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

var frameTypes = map[FrameType]string{
	FrameTypeReserved:        "RESERVED",
	FrameTypeSetup:           "SETUP",
	FrameTypeLease:           "LEASE",
	FrameTypeKeepalive:       "KEEPALIVE",
	FrameTypeRequestResponse: "REQUEST_RESPONSE",
	FrameTypeRequestFNF:      "REQUEST_FNF",
	FrameTypeRequestStream:   "REQUEST_STREAM",
	FrameTypeRequestChannel:  "REQUEST_CHANNEL",
	FrameTypeRequestN:        "REQUEST_N",
	FrameTypeCancel:          "CANCEL",
	FrameTypePayload:         "PAYLOAD",
	FrameTypeError:           "ERROR",
	FrameTypeMetadataPush:    "METADATA_PUSH",
	FrameTypeResume:          "RESUME",
	FrameTypeResumeOK:        "RESUME_OK",
	FrameTypeExt:             "EXT",
}

func (f FrameType) String() string {
	s, ok := frameTypes[f]
	if !ok {
		return "UNKNOWN"
	}
	return s
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
		foo = append(foo, "FRS)")
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
	// Release release resources of frame.
	Release()
	// Len returns length of frame.
	Len() int
	// Validate returns error if frame is invalid.
	Validate() error
	// SetHeader set frame header.
	SetHeader(h FrameHeader)
	// SetBody set frame body.
	SetBody(body *common.ByteBuff)
}

// BaseFrame is basic frame implementation.
type BaseFrame struct {
	header FrameHeader
	body   *common.ByteBuff
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

// Release release resources of frame.
func (p *BaseFrame) Release() {
	if p.body == nil {
		return
	}
	common.ReturnByteBuffer(p.body)
	p.body = nil
}

// NewBaseFrame returns a new BaseFrame.
func NewBaseFrame(h FrameHeader, body *common.ByteBuff) *BaseFrame {
	return &BaseFrame{
		header: h,
		body:   body,
	}
}

func (p *BaseFrame) trySeekMetadataLen(offset int) int {
	raw := p.body.Bytes()
	if offset > 0 {
		raw = raw[offset:]
	}
	if !p.header.Flag().Check(FlagMetadata) {
		return 0
	}
	if len(raw) < 3 {
		return -1
	}
	return common.NewUint24Bytes(raw).AsInt()
}

func (p *BaseFrame) trySliceMetadata(offset int) ([]byte, bool) {
	n := p.trySeekMetadataLen(offset)
	if n < 1 {
		return nil, false
	}
	return p.body.Bytes()[offset+3 : offset+3+n], true
}

func (p *BaseFrame) trySliceData(offset int) []byte {
	n := p.trySeekMetadataLen(offset)
	if n < 0 {
		return nil
	}
	if n == 0 {
		return p.body.Bytes()
	}
	return p.body.Bytes()[offset+n+3:]
}
