package rsocket

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

const (
	headerLen   = 6
	maxBuffSize = 16*1024*1024 + 3
)

var (
	errIncompleteHeader = errors.New("incomplete header")
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
	buf := make([]byte, 0, defaultConnTcpReadBuffSize)
	p.scanner.Buffer(buf, maxBuffSize)
	for p.scanner.Scan() {
		data := p.scanner.Bytes()[3:]
		if len(data) < headerLen {
			return errIncompleteHeader
		}
		if err := fn(data); err != nil {
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

// toError try convert something to error
func toError(err interface{}) error {
	if err == nil {
		return nil
	}
	switch v := err.(type) {
	case error:
		return v
	case string:
		return errors.New(v)
	default:
		return fmt.Errorf("%s", v)
	}
}
