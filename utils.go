package rsocket

import (
	"bufio"
	"io"
	"net"
	"strings"
)

const (
	headerLen       = 6
	defaultBuffSize = 16 * 1024
	maxBuffSize     = 16*1024*1024 + 3
)

type handleBytes = func(raw []byte) error

type frameDecoder interface {
	handle(fn handleBytes) error
}

type lengthBasedFrameDecoder struct {
	scanner *bufio.Scanner
}

func (p *lengthBasedFrameDecoder) handle(fn handleBytes) error {
	p.scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF {
			return
		}
		if len(data) < 3 {
			return
		}
		frameLength := newUint24Bytes(data).asInt()
		if frameLength < 1 {
			err = ErrInvalidFrameLength
			return
		}
		frameSize := frameLength + 3
		if frameSize <= len(data) {
			return frameSize, data[:frameSize], nil
		}
		return
	})
	buf := make([]byte, 0, defaultBuffSize)
	p.scanner.Buffer(buf, maxBuffSize)
	for p.scanner.Scan() {
		data := p.scanner.Bytes()[3:]
		if err := fn(data); err != nil {
			return err
		}
	}
	err := p.scanner.Err()
	if err == nil {
		return nil
	}
	if foo, ok := err.(*net.OpError); ok && strings.Contains(foo.Err.Error(), "use of closed network connection") {
		return nil
	}
	return err
}

func newLengthBasedFrameDecoder(r io.Reader) *lengthBasedFrameDecoder {
	return &lengthBasedFrameDecoder{
		scanner: bufio.NewScanner(r),
	}
}

func extractMetadataAndData(h header, raw []byte) (metadata []byte, data []byte) {
	if raw == nil {
		return nil, nil
	}
	if !h.Flag().Check(FlagMetadata) {
		data = raw[:]
		return
	}
	l := newUint24Bytes(raw).asInt()
	metadata = raw[3 : l+3]
	data = raw[l+3:]
	return
}
