package protocol

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	frameHeaderLength int = 6
)

var (
	byteOrder = binary.BigEndian
)

type lengthBasedFrameDecoder struct {
	scanner *bufio.Scanner
}

func (p *lengthBasedFrameDecoder) Handle(fn FrameHandler) error {
	p.scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF {
			return
		}
		if len(data) < 3 {
			return
		}
		frameLength := readUint24(data, 0)
		if frameLength < 1 {
			err = fmt.Errorf("bad frame length: %d", frameLength)
			return
		}
		frameSize := frameLength + 3
		if frameSize <= len(data) {
			return frameSize, data[:frameSize], nil
		}
		return
	})
	for p.scanner.Scan() {
		data := p.scanner.Bytes()
		f := Frame(data[3:])
		if err := fn(f); err != nil {
			return err
		}
	}
	return p.scanner.Err()
}

func newLengthBasedFrameDecoder(r io.Reader) *lengthBasedFrameDecoder {
	return &lengthBasedFrameDecoder{
		scanner: bufio.NewScanner(r),
	}
}

func readUint24(bs []byte, offset int) int {
	return int(bs[offset])<<16 + int(bs[offset+1])<<8 + int(bs[offset+2])
}

func writeUint24(n int) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(n))
	return b[1:]
}
