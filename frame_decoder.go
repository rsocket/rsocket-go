package rsocket

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

const (
	headerLen       = 6
	defaultBuffSize = 64 * 1024
	maxBuffSize     = 16*1024*1024 + 3
)

type FrameHandler = func(h *Header, raw []byte) error

type FrameDecoder interface {
	Handle(ctx context.Context, fn FrameHandler) error
}

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
	buf := make([]byte, 0, defaultBuffSize)
	p.scanner.Buffer(buf, maxBuffSize)
	for p.scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			p.scanner.Bytes()
			data := p.scanner.Bytes()[3:]
			h, err := asHeader(data)
			if err != nil {
				return err
			}
			if err := fn(h, data); err != nil {
				return err
			}
		}
	}
	if err := p.scanner.Err(); err != nil {
		if foo, ok := err.(*net.OpError); ok && strings.EqualFold(foo.Err.Error(), "use of closed network connection") {
			return nil
		}
		log.Println("scanner err:", err)
		return err
	}
	return nil
}

func newLengthBasedFrameDecoder(r io.Reader) *lengthBasedFrameDecoder {
	return &lengthBasedFrameDecoder{
		scanner: bufio.NewScanner(r),
	}
}
