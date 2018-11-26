package protocol

func readUint24WithOffset(bs []byte, offset int) int {
	return int(bs[offset])<<16 + int(bs[offset+1])<<8 + int(bs[offset+2])
}

func readUint24(bs []byte) int {
	return readUint24WithOffset(bs, 0)
}

func readMetadataAndPayload(input []byte) (metadata []byte, payload []byte, err error) {
	defer func() {
		if e, ok := recover().(error); ok {
			err = e
		}
	}()
	metadataLength := readUint24(input)
	metadata = input[3 : 3+metadataLength]
	payload = input[3+metadataLength:]
	return
}
