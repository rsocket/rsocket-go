package core

import (
	"encoding/binary"
	"io"
	"strconv"
	"strings"
)

const (
	// FrameHeaderLen is len of header.
	FrameHeaderLen = 6
)

// FrameHeader is the header fo a RSocket frame.
// RSocket frames begin with a RSocket frame header.
// It includes StreamID, FrameType and Flags.
type FrameHeader [FrameHeaderLen]byte

func (h FrameHeader) String() string {
	bu := strings.Builder{}
	bu.WriteString("FrameHeader{id=")
	bu.WriteString(strconv.FormatUint(uint64(h.StreamID()), 10))
	bu.WriteString(",type=")
	bu.WriteString(h.Type().String())
	bu.WriteString(",flag=")
	bu.WriteString(h.Flag().String())
	bu.WriteByte('}')
	return bu.String()
}

// Resumable returns true if frame supports resume.
func (h FrameHeader) Resumable() bool {
	switch h.Type() {
	case FrameTypeRequestChannel, FrameTypeRequestStream, FrameTypeRequestResponse, FrameTypeRequestFNF, FrameTypeRequestN, FrameTypeCancel, FrameTypeError, FrameTypePayload:
		return true
	default:
		return false
	}
}

// WriteTo writes frame header to a writer.
func (h FrameHeader) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(h[:])
	return int64(n), err
}

// StreamID returns StreamID.
func (h FrameHeader) StreamID() uint32 {
	return binary.BigEndian.Uint32(h[:4])
}

// Type returns frame type.
func (h FrameHeader) Type() FrameType {
	return FrameType((h.n() & 0xFC00) >> 10)
}

// Flag returns flag of a frame.
func (h FrameHeader) Flag() FrameFlag {
	return FrameFlag(h.n() & 0x03FF)
}

// Bytes returns raw frame header bytes.
func (h FrameHeader) Bytes() []byte {
	return h[:]
}

func (h FrameHeader) n() uint16 {
	return binary.BigEndian.Uint16(h[4:])
}

// NewFrameHeader returns a new frame header.
func NewFrameHeader(streamID uint32, frameType FrameType, fg FrameFlag) FrameHeader {
	var h [FrameHeaderLen]byte
	binary.BigEndian.PutUint32(h[:], streamID)
	binary.BigEndian.PutUint16(h[4:], uint16(frameType)<<10|uint16(fg))
	return h
}

// WriteFrameHeader writes frame header into a io.Writer.
func WriteFrameHeader(w io.Writer, streamID uint32, frameType FrameType, fg FrameFlag) error {
	if err := binary.Write(w, binary.BigEndian, streamID); err != nil {
		return err
	}
	return binary.Write(w, binary.BigEndian, uint16(frameType)<<10|uint16(fg))
}

// ResetFrameHeader resets the original frame header bytes.
func ResetFrameHeader(b []byte, streamID uint32, frameType FrameType, fg FrameFlag) {
	binary.BigEndian.PutUint32(b, streamID)
	binary.BigEndian.PutUint16(b[4:], uint16(frameType)<<10|uint16(fg))
}

// ParseFrameHeader parse a header from bytes.
func ParseFrameHeader(bs []byte) FrameHeader {
	_ = bs[FrameHeaderLen-1]
	var bb [FrameHeaderLen]byte
	copy(bb[:], bs[:FrameHeaderLen])
	return bb
}
