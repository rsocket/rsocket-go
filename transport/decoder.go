package transport

import (
	"bufio"
	"errors"
	"io"

	"github.com/rsocket/rsocket-go/common"
	"github.com/rsocket/rsocket-go/framing"
)

const (
	lengthFieldSize = 3
	maxBuffSize     = 16*1024*1024 + lengthFieldSize
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
		if len(data) < lengthFieldSize {
			return
		}
		frameLength := common.NewUint24Bytes(data).AsInt()
		if frameLength < 1 {
			err = common.ErrInvalidFrameLength
			return
		}
		frameSize := frameLength + lengthFieldSize
		if frameSize <= len(data) {
			return frameSize, data[:frameSize], nil
		}
		return
	})
	buf := make([]byte, 0, common.DefaultTCPReadBuffSize)
	p.scanner.Buffer(buf, maxBuffSize)
	for p.scanner.Scan() {
		data := p.scanner.Bytes()[lengthFieldSize:]
		if len(data) < framing.HeaderLen {
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
