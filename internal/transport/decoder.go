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
	minBuffSize     = 4 * 1024
	maxBuffSize     = 16*1024*1024 + lengthFieldSize
)

// ErrIncompleteHeader is error of incomplete header.
var ErrIncompleteHeader = errors.New("incomplete frame header")

// LengthBasedFrameDecoder defines a decoder for decoding frames which have a header of length.
type LengthBasedFrameDecoder bufio.Scanner

// Read reads next raw frame in bytes.
func (p *LengthBasedFrameDecoder) Read() (raw []byte, err error) {
	scanner := (*bufio.Scanner)(p)
	if !scanner.Scan() {
		err = scanner.Err()
		if err == nil || isClosedErr(err) {
			err = io.EOF
		}
		return
	}
	raw = scanner.Bytes()[lengthFieldSize:]
	if len(raw) < framing.HeaderLen {
		err = ErrIncompleteHeader
	}
	return
}

func doSplit(data []byte, atEOF bool) (advance int, token []byte, err error) {
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

// NewLengthBasedFrameDecoder creates a new frame decoder.
func NewLengthBasedFrameDecoder(r io.Reader) *LengthBasedFrameDecoder {
	scanner := bufio.NewScanner(r)
	scanner.Split(doSplit)
	buf := make([]byte, 0, minBuffSize)
	scanner.Buffer(buf, maxBuffSize)
	return (*LengthBasedFrameDecoder)(scanner)
}
