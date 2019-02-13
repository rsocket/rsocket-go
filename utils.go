package rsocket

import (
	"bufio"
	"context"
	"io"
	"net"
	"strings"
)

const (
	headerLen       = 6
	defaultBuffSize = 64 * 1024
	maxBuffSize     = 16*1024*1024 + 3
)

type frameHandler = func(raw []byte) error

type frameDecoder interface {
	handle(ctx context.Context, fn frameHandler) error
}

type lengthBasedFrameDecoder struct {
	scanner *bufio.Scanner
}

func (p *lengthBasedFrameDecoder) handle(ctx context.Context, fn frameHandler) error {
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
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			data := p.scanner.Bytes()[3:]
			if err := fn(data); err != nil {
				return err
			}
		}
	}
	if err := p.scanner.Err(); err != nil {
		if foo, ok := err.(*net.OpError); ok && strings.EqualFold(foo.Err.Error(), "use of closed network connection") {
			return nil
		}
		return err
	}
	return nil
}

func newLengthBasedFrameDecoder(r io.Reader) *lengthBasedFrameDecoder {
	return &lengthBasedFrameDecoder{
		scanner: bufio.NewScanner(r),
	}
}

func extractMetadataAndData(h header, raw []byte) (metadata []byte, data []byte) {
	if !h.Flag().Check(FlagMetadata) {
		data = raw[:]
		return
	}
	l := newUint24Bytes(raw).asInt()
	metadata = raw[3 : l+3]
	data = raw[l+3:]
	return
}

type ErrorCode uint32

func (p ErrorCode) String() string {
	switch p {
	case ErrorCodeInvalidSetup:
		return "INVALID_SETUP"
	case ErrorCodeUnsupportedSetup:
		return "UNSUPPORTED_SETUP"
	case ErrorCodeRejectedSetup:
		return "REJECTED_SETUP"
	case ErrorCodeRejectedResume:
		return "REJECTED_RESUME"
	case ErrorCodeConnectionError:
		return "CONNECTION_ERROR"
	case ErrorCodeConnectionClose:
		return "CONNECTION_CLOSE"
	case ErrorCodeApplicationError:
		return "APPLICATION_ERROR"
	case ErrorCodeRejected:
		return "REJECTED"
	case ErrorCodeCanceled:
		return "CANCELED"
	case ErrorCodeInvalid:
		return "INVALID"
	case 0x00000000:
		return "RESERVED"
	case 0xFFFFFFFF:
		return "RESERVED"
	default:
		return "UNKNOWN"
	}
}

const (
	ErrorCodeInvalidSetup     ErrorCode = 0x00000001
	ErrorCodeUnsupportedSetup ErrorCode = 0x00000002
	ErrorCodeRejectedSetup    ErrorCode = 0x00000003
	ErrorCodeRejectedResume   ErrorCode = 0x00000004
	ErrorCodeConnectionError  ErrorCode = 0x00000101
	ErrorCodeConnectionClose  ErrorCode = 0x00000102
	ErrorCodeApplicationError ErrorCode = 0x00000201
	ErrorCodeRejected         ErrorCode = 0x00000202
	ErrorCodeCanceled         ErrorCode = 0x00000203
	ErrorCodeInvalid          ErrorCode = 0x00000204
)
