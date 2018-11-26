package protocol

func readUint24WithOffset(bs []byte, offset int) int {
	return int(bs[offset])<<16 + int(bs[offset+1])<<8 + int(bs[offset+2])
}

func readUint24(bs []byte) int {
	return readUint24WithOffset(bs, 0)
}
