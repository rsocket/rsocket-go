package rsocket

import (
	"io"
	"strings"
)

type rFrameType uint8

const (
	tReserved        rFrameType = 0x00
	tSetup           rFrameType = 0x01
	tLease           rFrameType = 0x02
	tKeepalive       rFrameType = 0x03
	tRequestResponse rFrameType = 0x04
	tRequestFNF      rFrameType = 0x05
	tRequestStream   rFrameType = 0x06
	tRequestChannel  rFrameType = 0x07
	tRequestN        rFrameType = 0x08
	tCancel          rFrameType = 0x09
	tPayload         rFrameType = 0x0A
	tError           rFrameType = 0x0B
	tMetadataPush    rFrameType = 0x0C
	tResume          rFrameType = 0x0D
	tResumeOK        rFrameType = 0x0E
	tExt             rFrameType = 0x3F
)

func (f rFrameType) String() string {
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

type rFlags uint16

func (f rFlags) String() string {
	foo := make([]string, 0)
	if f.Check(flagNext) {
		foo = append(foo, "N")
	}
	if f.Check(flagComplete) {
		foo = append(foo, "CL")
	}
	if f.Check(flagFollow) {
		foo = append(foo, "FRS)")
	}
	if f.Check(flagMetadata) {
		foo = append(foo, "M")
	}
	if f.Check(flagIgnore) {
		foo = append(foo, "I")
	}
	return strings.Join(foo, "|")
}

const (
	flagNext rFlags = 1 << (5 + iota)
	flagComplete
	flagFollow
	flagMetadata
	flagIgnore

	flagResume  = flagFollow
	flagLease   = flagComplete
	flagRespond = flagFollow
)

func (f rFlags) Check(mask rFlags) bool {
	return mask&f == mask
}

func newFlags(flags ...rFlags) rFlags {
	var fg rFlags
	for _, it := range flags {
		fg |= it
	}
	return fg
}

// Frame is a single message containing a request, response, or protocol processing.
type Frame interface {
	io.WriterTo
	// Header returns frame header.
	Header() header
	// Release release resources of frame.
	Release()
	// Len returns length of frame.
	Len() int

	validate() error
	setHeader(h header)
}

type baseFrame struct {
	header header
	body   *rByteBuffer
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
	if !p.header.Flag().Check(flagMetadata) {
		return 0
	}
	if len(raw) < 3 {
		return -1
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
