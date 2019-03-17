// +build gofuzz

//go:generate GO111MODULE=off go-fuzz-build github.com/rsocket/rsocket-go/

package rsocket

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/rsocket/rsocket-go/framing"
	"github.com/rsocket/rsocket-go/common"
)

func Fuzz(data []byte) int {
	buf := bytes.NewBuffer(data)
	decoder := newLengthBasedFrameDecoder(buf)

	err := decoder.handle(func(raw []byte) error {
		h := framing.ParseFrameHeader(data)
		bf := common.BorrowByteBuffer()
		f := framing.NewBaseFrame(h, bf)

		var frame framing.Frame

		switch f.Header().Type() {
		case framing.FrameTypeSetup:
			frame = &framing.FrameSetup{BaseFrame: f}
		case framing.FrameTypeKeepalive:
			frame = &framing.FrameKeepalive{BaseFrame: f}
		case framing.FrameTypeRequestResponse:
			frame = &framing.FrameRequestResponse{BaseFrame: f}
		case framing.FrameTypeRequestFNF:
			frame = &framing.FrameFNF{BaseFrame: f}
		case framing.FrameTypeRequestStream:
			frame = &framing.FrameRequestStream{BaseFrame: f}
		case framing.FrameTypeRequestChannel:
			frame = &framing.FrameRequestChannel{BaseFrame: f}
		case framing.FrameTypeCancel:
			frame = &framing.FrameCancel{BaseFrame: f}
		case framing.FrameTypePayload:
			frame = &framing.FramePayload{BaseFrame: f}
		case framing.FrameTypeMetadataPush:
			frame = &framing.FrameMetadataPush{BaseFrame: f}
		case framing.FrameTypeError:
			frame = &framing.FrameError{BaseFrame: f}
		case framing.FrameTypeRequestN:
			frame = &framing.FrameRequestN{BaseFrame: f}
		default:
			return ErrInvalidFrame
		}

		// release resource borrowed
		defer f.Release()

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
