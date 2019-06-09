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

var ErrIncompleteHeader = errors.New("incomplete frame header")

type LengthBasedFrameDecoder bufio.Scanner

func (p *LengthBasedFrameDecoder) Read() (raw []byte, err error) {
	scanner := (*bufio.Scanner)(p)
	if !scanner.Scan() {
		err = scanner.Err()
		if err == nil {
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

func NewLengthBasedFrameDecoder(r io.Reader) *LengthBasedFrameDecoder {
	scanner := bufio.NewScanner(r)
	scanner.Split(doSplit)
	buf := make([]byte, 0, tcpReadBuffSize)
	scanner.Buffer(buf, maxBuffSize)
	return (*LengthBasedFrameDecoder)(scanner)
}
