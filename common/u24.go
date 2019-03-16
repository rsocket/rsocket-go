package common

import (
	"encoding/binary"
	"io"
)

// Uint24 is 3 bytes unsigned integer.
type Uint24 [3]byte

// Bytes returns bytes encoded.
func (p Uint24) Bytes() []byte {
	return p[:]
}

// WriteTo encode and write bytes to a writer.
func (p Uint24) WriteTo(w io.Writer) (int64, error) {
	wrote, err := w.Write(p[:])
	return int64(wrote), err
}

// AsInt converts to int.
func (p Uint24) AsInt() int {
	return int(p[0])<<16 + int(p[1])<<8 + int(p[2])
}

// NewUint24 returns a new uint24.
func NewUint24(n int) Uint24 {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], uint32(n))
	return [3]byte{b[1], b[2], b[3]}
}

// NewUint24Bytes returns a new uint24 from bytes.
func NewUint24Bytes(bs []byte) Uint24 {
	_ = bs[2]
	return [3]byte{bs[0], bs[1], bs[2]}
}
