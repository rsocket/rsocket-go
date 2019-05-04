package framing

import (
	"github.com/rsocket/rsocket-go/internal/common"
)

// CalcPayloadFrameSize returns payload frame size.
func CalcPayloadFrameSize(data, metadata []byte) int {
	size := HeaderLen + len(data)
	if n := len(metadata); n > 0 {
		size += 3 + n
	}
	return size
}

func NewFromBase(f *BaseFrame) (frame Frame, err error) {
	switch f.header.Type() {
	case FrameTypeSetup:
		frame = &FrameSetup{BaseFrame: f}
	case FrameTypeKeepalive:
		frame = &FrameKeepalive{BaseFrame: f}
	case FrameTypeRequestResponse:
		frame = &FrameRequestResponse{BaseFrame: f}
	case FrameTypeRequestFNF:
		frame = &FrameFNF{BaseFrame: f}
	case FrameTypeRequestStream:
		frame = &FrameRequestStream{BaseFrame: f}
	case FrameTypeRequestChannel:
		frame = &FrameRequestChannel{BaseFrame: f}
	case FrameTypeCancel:
		frame = &FrameCancel{BaseFrame: f}
	case FrameTypePayload:
		frame = &FramePayload{BaseFrame: f}
	case FrameTypeMetadataPush:
		frame = &FrameMetadataPush{BaseFrame: f}
	case FrameTypeError:
		frame = &FrameError{BaseFrame: f}
	case FrameTypeRequestN:
		frame = &FrameRequestN{BaseFrame: f}
	case FrameTypeLease:
		frame = &FrameLease{BaseFrame: f}
	case FrameTypeResume:
		frame = &FrameResume{BaseFrame: f}
	case FrameTypeResumeOK:
		frame = &FrameResumeOK{BaseFrame: f}
	default:
		err = common.ErrInvalidFrame
	}
	return
}
