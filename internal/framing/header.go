package framing

import (
	"encoding/binary"
	"io"
	"strconv"
	"strings"
)

const (
	// HeaderLen is len of header.
	HeaderLen = 6
)

// Header is the header fo a RSocket frame.
// RSocket frames begin with a RSocket Frame Header.
// It includes StreamID, FrameType and Flags.
type Header [HeaderLen]byte

func (h Header) String() string {
	bu := strings.Builder{}
	bu.WriteString("Header{id=")
	bu.WriteString(strconv.FormatUint(uint64(h.StreamID()), 10))
	bu.WriteString(",type=")
	bu.WriteString(h.Type().String())
	bu.WriteString(",flag=")
	bu.WriteString(h.Flag().String())
	bu.WriteByte('}')
	return bu.String()
}

// Resumable returns true if frame supports resume.
func (h Header) Resumable() bool {
	switch h.Type() {
	case FrameTypeRequestChannel, FrameTypeRequestStream, FrameTypeRequestResponse, FrameTypeRequestFNF, FrameTypeRequestN, FrameTypeCancel, FrameTypeError, FrameTypePayload:
		return true
	default:
		return false
	}
}

// WriteTo writes frame header to a writer.
func (h Header) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(h[:])
	return int64(n), err
}

// StreamID returns StreamID.
func (h Header) StreamID() uint32 {
	return binary.BigEndian.Uint32(h[:4])
}

// Type returns frame type.
func (h Header) Type() FrameType {
	return FrameType((h.n() & 0xFC00) >> 10)
}

// Flag returns flag of a frame.
func (h Header) Flag() FrameFlag {
	return FrameFlag(h.n() & 0x03FF)
}

func (h Header) Bytes() []byte {
	return h[:]
}

func (h Header) n() uint16 {
	return binary.BigEndian.Uint16(h[4:])
}

// NewFrameHeader returns a new frame header.
func NewFrameHeader(streamID uint32, frameType FrameType, fg FrameFlag) Header {
	var h [HeaderLen]byte
	binary.BigEndian.PutUint32(h[:], streamID)
	binary.BigEndian.PutUint16(h[4:], uint16(frameType)<<10|uint16(fg))
	return h

}

// ParseFrameHeader parse a header from bytes.
func ParseFrameHeader(bs []byte) Header {
	_ = bs[HeaderLen-1]
	var bb [HeaderLen]byte
	copy(bb[:], bs[:HeaderLen])
	return bb
}
