package rsocket

import (
	"io"
	"strings"
)

type FrameType uint8

const (
	tReserved        FrameType = 0x00
	tSetup           FrameType = 0x01
	tLease           FrameType = 0x02
	tKeepalive       FrameType = 0x03
	tRequestResponse FrameType = 0x04
	tRequestFNF      FrameType = 0x05
	tRequestStream   FrameType = 0x06
	tRequestChannel  FrameType = 0x07
	tRequestN        FrameType = 0x08
	tCancel          FrameType = 0x09
	tPayload         FrameType = 0x0A
	tError           FrameType = 0x0B
	tMetadataPush    FrameType = 0x0C
	tResume          FrameType = 0x0D
	tResumeOK        FrameType = 0x0E
	tExt             FrameType = 0x3F
)

func (f FrameType) String() string {
	switch f {
	case tReserved:
		return "RESERVED"
	case tSetup:
		return "SETUP"
	case tLease:
		return "LEASE"
	case tKeepalive:
		return "KEEPALIVE"
	case tRequestResponse:
		return "REQUEST_RESPONSE"
	case tRequestFNF:
		return "REQUEST_FNF"
	case tRequestStream:
		return "REQUEST_STREAM"
	case tRequestChannel:
		return "REQUEST_CHANNEL"
	case tRequestN:
		return "REQUEST_N"
	case tCancel:
		return "CANCEL"
	case tPayload:
		return "PAYLOAD"
	case tError:
		return "ERROR"
	case tMetadataPush:
		return "METADATA_PUSH"
	case tResume:
		return "RESUME"
	case tResumeOK:
		return "RESUME_OK"
	case tExt:
		return "EXT"
	default:
		return "UNKNOWN"
	}
}

type Flags uint16

func (f Flags) String() string {
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

func newFlags(flags ...Flags) Flags {
	var fg Flags
	for _, it := range flags {
		fg |= it
	}
	return fg
}

type Frame interface {
	io.WriterTo
	setHeader(h header)
	Header() header
	Release()
	Len() int
}

type baseFrame struct {
	header header
	body   *ByteBuffer
}

func (p *baseFrame) Header() header {
	return p.header
}

func (p *baseFrame) setHeader(h header) {
	p.header = h
}

func (p *baseFrame) Len() int {
	return headerLen + p.body.Len()
}

func (p *baseFrame) WriteTo(w io.Writer) (n int64, err error) {
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

func (p *baseFrame) Release() {
	if p.body == nil {
		return
	}
	returnByteBuffer(p.body)
	p.body = nil
}

func (p *baseFrame) trySeekMetadataLen(offset int) int {
	raw := p.body.Bytes()
	if offset > 0 {
		raw = raw[offset:]
	}
	if raw == nil {
		return -1
	}
	if !p.header.Flag().Check(FlagMetadata) {
		return 0
	}
	return newUint24Bytes(raw).asInt()
}

func (p *baseFrame) trySliceMetadata(offset int) []byte {
	n := p.trySeekMetadataLen(offset)
	if n < 1 {
		return nil
	}
	return p.body.Bytes()[offset+3 : offset+3+n]
}

func (p *baseFrame) trySliceData(offset int) []byte {
	n := p.trySeekMetadataLen(offset)
	if n < 0 {
		return nil
	}
	if n == 0 {
		return p.body.Bytes()
	}
	return p.body.Bytes()[offset+n+3:]
}
