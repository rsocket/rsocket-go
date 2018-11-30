package rsocket

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	frameHeaderLength int = 6
)

type lengthBasedFrameDecoder struct {
	scanner *bufio.Scanner
}

func (p *lengthBasedFrameDecoder) Handle(ctx context.Context, fn FrameHandler) error {
	p.scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF {
			return
		}
		if len(data) < 3 {
			return
		}
		frameLength := decodeU24(data, 0)
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
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			p.scanner.Bytes()
			data := p.scanner.Bytes()
			clone := make([]byte, len(data)-3)
			copy(clone, data[3:])
			h, err := asHeader(clone)
			if err != nil {
				return err
			}
			if err := fn(h, clone); err != nil {
				return err
			}
		}
	}
	return p.scanner.Err()
}

func newLengthBasedFrameDecoder(r io.Reader) *lengthBasedFrameDecoder {
	return &lengthBasedFrameDecoder{
		scanner: bufio.NewScanner(r),
	}
}

func decodeU24(bs []byte, offset int) int {
	return int(bs[offset])<<16 + int(bs[offset+1])<<8 + int(bs[offset+2])
}

func encodeU24(n int) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(n))
	return b[1:]
}

func sliceMetadataAndData(header *Header, raw []byte, offset int) (metadata []byte, data []byte) {
	if !header.Flags().Check(FlagMetadata) {
		return nil, raw[offset:]
	}
	l := decodeU24(raw, offset)
	offset += 3
	return raw[offset : offset+l], raw[offset+l:]
}

func sliceMetadata(header *Header, raw []byte, offset int) []byte {
	if !header.Flags().Check(FlagMetadata) {
		return nil
	}
	l := decodeU24(raw, offset)
	offset += 3
	return raw[offset : offset+l]
}

func sliceData(header *Header, raw []byte, offset int) []byte {
	if header.Flags().Check(FlagMetadata) {
		offset += 3 + decodeU24(raw, offset)
	}
	return raw[offset:]
}
