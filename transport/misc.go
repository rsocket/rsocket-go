package transport

import (
	"bytes"

	"github.com/rsocket/rsocket-go/common"
	"github.com/rsocket/rsocket-go/framing"
)

var keepaliveBytes []byte

func init() {
	frame := framing.NewFrameKeepalive(0, nil, true)
	bf := &bytes.Buffer{}
	_, err := common.NewUint24(frame.Len()).WriteTo(bf)
	if err != nil {
		panic(err)
	}
	_, err = frame.WriteTo(bf)
	if err != nil {
		panic(err)
	}
	keepaliveBytes = bf.Bytes()
	frame.Release()
}

func getKeepaliveBytes(hasLengthField bool) []byte {
	if hasLengthField {
		return keepaliveBytes
	}
	return keepaliveBytes[lengthFieldSize:]
}

func unwindFrame(f *framing.BaseFrame) (frame framing.Frame, err error) {
	frameType := f.Header().Type()
	switch frameType {
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
	case framing.FrameTypeLease:
		frame = &framing.FrameLease{BaseFrame: f}
	default:
		err = common.ErrInvalidFrame
	}
	return
}
