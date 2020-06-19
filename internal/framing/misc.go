package framing

import (
	"io"

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

// FromRawFrame creates a frame from a RawFrame.
func FromRawFrame(f *RawFrame) (frame Frame, err error) {
	switch f.header.Type() {
	case FrameTypeSetup:
		frame = &SetupFrame{RawFrame: f}
	case FrameTypeKeepalive:
		frame = &KeepaliveFrame{RawFrame: f}
	case FrameTypeRequestResponse:
		frame = &RequestResponseFrame{RawFrame: f}
	case FrameTypeRequestFNF:
		frame = &FireAndForgetFrame{RawFrame: f}
	case FrameTypeRequestStream:
		frame = &RequestStreamFrame{RawFrame: f}
	case FrameTypeRequestChannel:
		frame = &RequestChannelFrame{RawFrame: f}
	case FrameTypeCancel:
		frame = &CancelFrame{RawFrame: f}
	case FrameTypePayload:
		frame = &PayloadFrame{RawFrame: f}
	case FrameTypeMetadataPush:
		frame = &MetadataPushFrame{RawFrame: f}
	case FrameTypeError:
		frame = &ErrorFrame{RawFrame: f}
	case FrameTypeRequestN:
		frame = &RequestNFrame{RawFrame: f}
	case FrameTypeLease:
		frame = &LeaseFrame{RawFrame: f}
	case FrameTypeResume:
		frame = &ResumeFrame{RawFrame: f}
	case FrameTypeResumeOK:
		frame = &ResumeOKFrame{RawFrame: f}
	default:
		err = common.ErrInvalidFrame
	}
	return
}

func writePayload(w io.Writer, data []byte, metadata []byte) (n int64, err error) {
	if l := len(metadata); l > 0 {
		var wrote int64
		u := common.MustNewUint24(l)
		wrote, err = u.WriteTo(w)
		if err != nil {
			return
		}
		n += wrote

		var v int
		v, err = w.Write(metadata)
		if err != nil {
			return
		}
		n += int64(v)
	}

	if l := len(data); l > 0 {
		var v int
		v, err = w.Write(data)
		if err != nil {
			return
		}
		n += int64(v)
	}
	return
}
