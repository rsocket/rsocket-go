package framing

import (
	"encoding/binary"
	"fmt"
	"io"
)

const (
	// HeaderLen is len of header.
	HeaderLen = 6
)

// FrameHeader is the header fo a RSocke frame.
// RSocket frames begin with a RSocket Frame Header.
// It includes StreamID, FrameType and Flags.
type FrameHeader [HeaderLen]byte

func (p FrameHeader) String() string {
	return fmt.Sprintf("FrameHeader{id=%d,type=%s,flag=%s}", p.StreamID(), p.Type(), p.Flag())
}

// WriteTo writes frame header to a writer.
func (p FrameHeader) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(p[:])
	return int64(n), err
}

// StreamID returns StreamID.
func (p FrameHeader) StreamID() uint32 {
	return binary.BigEndian.Uint32(p[:4])
}

// Type returns frame type.
func (p FrameHeader) Type() FrameType {
	return FrameType((p.n() & 0xFC00) >> 10)
}

// Flag returns flag of a frame.
func (p FrameHeader) Flag() FrameFlag {
	return FrameFlag(p.n() & 0x03FF)
}

func (p FrameHeader) n() uint16 {
	return binary.BigEndian.Uint16(p[4:])
}

// NewFrameHeader returns a new frame header.
func NewFrameHeader(streamID uint32, frameType FrameType, flags ...FrameFlag) FrameHeader {
	fg := newFlags(flags...)
	var h [HeaderLen]byte
	binary.BigEndian.PutUint32(h[:], streamID)
	binary.BigEndian.PutUint16(h[4:], uint16(frameType)<<10|uint16(fg))
	return h
}

// ParseFrameHeader parse a header from bytes.
func ParseFrameHeader(bs []byte) FrameHeader {
	_ = bs[HeaderLen-1]
	var bb [HeaderLen]byte
	copy(bb[:], bs[:HeaderLen])
	return FrameHeader(bb)
}
