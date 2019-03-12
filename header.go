package rsocket

import (
	"encoding/binary"
	"fmt"
	"io"
)

type header [headerLen]byte

func (p header) String() string {
	return fmt.Sprintf("header{id=%d,type=%s,flag=%s}", p.StreamID(), p.Type(), p.Flag())
}

func (p header) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(p[:])
	return int64(n), err
}

func (p header) StreamID() uint32 {
	return binary.BigEndian.Uint32(p[:4])
}

func (p header) Type() rFrameType {
	return rFrameType((p.n() & 0xFC00) >> 10)
}

func (p header) Flag() rFlags {
	return rFlags(p.n() & 0x03FF)
}

func (p header) n() uint16 {
	return binary.BigEndian.Uint16(p[4:])
}

func createHeader(streamID uint32, frameType rFrameType, flags ...rFlags) header {
	fg := newFlags(flags...)
	var h [headerLen]byte
	binary.BigEndian.PutUint32(h[:], streamID)
	binary.BigEndian.PutUint16(h[4:], uint16(frameType)<<10|uint16(fg))
	return h
}

func parseHeaderBytes(bs []byte) header {
	_ = bs[headerLen-1]
	var bb [headerLen]byte
	copy(bb[:], bs[:headerLen])
	return header(bb)
}
