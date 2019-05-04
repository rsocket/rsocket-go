package transport

import (
	"bufio"
	"errors"
	"io"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/framing"
)

const (
	lengthFieldSize = 3
	maxBuffSize     = 16*1024*1024 + lengthFieldSize
)

var ErrIncompleteHeader = errors.New("incomplete header")

type FrameDecoder interface {
	Handle(fn func(raw []byte) error) error
}

type lengthBasedFrameDecoder struct {
	scanner *bufio.Scanner
}

func (p *lengthBasedFrameDecoder) Handle(fn func([]byte) error) error {
	p.scanner.Split(split)
	buf := make([]byte, 0, common.DefaultTCPReadBuffSize)
	p.scanner.Buffer(buf, maxBuffSize)
	for p.scanner.Scan() {
		data := p.scanner.Bytes()[lengthFieldSize:]
		if len(data) < framing.HeaderLen {
			return ErrIncompleteHeader
		}
		if err := fn(data); err != nil {
			return err
		}
	}
	return p.scanner.Err()
}

func split(data []byte, atEOF bool) (advance int, token []byte, err error) {
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
}

func NewLengthBasedFrameDecoder(r io.Reader) FrameDecoder {
	return &lengthBasedFrameDecoder{
		scanner: bufio.NewScanner(r),
	}
}
