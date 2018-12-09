package rsocket

import "encoding/binary"

func decodeU24(bs []byte, offset int) int {
	return int(bs[offset])<<16 + int(bs[offset+1])<<8 + int(bs[offset+2])
}

func encodeU24(n int) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(n))
	return b[1:]
}
