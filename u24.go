package rsocket

import (
	"encoding/binary"
	"io"
)

type uint24 [3]byte

func (p uint24) Bytes() []byte {
	return p[:]
}

func (p uint24) WriteTo(w io.Writer) (int64, error) {
	wrote, err := w.Write(p[:])
	return int64(wrote), err
}

func (p uint24) asInt() int {
	return int(p[0])<<16 + int(p[1])<<8 + int(p[2])
}

func newUint24(n int) uint24 {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], uint32(n))
	return [3]byte{b[1], b[2], b[3]}
}

func newUint24Bytes(bs []byte) uint24 {
	_ = bs[2]
	return [3]byte{bs[0], bs[1], bs[2]}
}
