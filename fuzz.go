// +build gofuzz

//go:generate GO111MODULE=off go-fuzz-build github.com/rsocket/rsocket-go/

package rsocket

import (
	"bytes"
	"errors"
	"fmt"
)

func Fuzz(data []byte) int {
	buf := bytes.NewBuffer(data)
	decoder := newLengthBasedFrameDecoder(buf)

	err := decoder.handle(func(raw []byte) error {
		h := parseHeaderBytes(data)
		bf := borrowByteBuffer()
		f := &baseFrame{h, bf}

		var frame Frame

		switch f.header.Type() {
		case tSetup:
			frame = &frameSetup{f}
		case tKeepalive:
			frame = &frameKeepalive{f}
		case tRequestResponse:
			frame = &frameRequestResponse{f}
		case tRequestFNF:
			frame = &frameFNF{f}
		case tRequestStream:
			frame = &frameRequestStream{f}
		case tRequestChannel:
			frame = &frameRequestChannel{f}
		case tCancel:
			frame = &frameCancel{f}
		case tPayload:
			frame = &framePayload{f}
		case tMetadataPush:
			frame = &frameMetadataPush{f}
		case tError:
			frame = &frameError{f}
		case tRequestN:
			frame = &frameRequestN{f}
		default:
			return ErrInvalidFrame
		}

		switch f := frame.(type) {
		case fmt.Stringer:
			s := f.String()

			if len(s) > 0 {
				return nil
			}
		case error:
			e := f.Error()

			if len(e) > 0 {
				return nil
			}
		default:
			panic("unreachable")
		}

		return errors.New("???")
	})

	if err == nil || err == ErrInvalidFrame {
		return 0
	}

	return 1
}
