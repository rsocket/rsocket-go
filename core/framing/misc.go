package framing

import (
	"io"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

// CalcPayloadFrameSize returns payload frame size.
func CalcPayloadFrameSize(data, metadata []byte) int {
	size := core.FrameHeaderLen + len(data)
	if n := len(metadata); n > 0 {
		size += 3 + n
	}
	return size
}

// FromRawFrame creates a frame from a RawFrame.
func FromRawFrame(f *RawFrame) (frame core.Frame, err error) {
	switch f.header.Type() {
	case core.FrameTypeSetup:
		frame = &SetupFrame{RawFrame: f}
	case core.FrameTypeKeepalive:
		frame = &KeepaliveFrame{RawFrame: f}
	case core.FrameTypeRequestResponse:
		frame = &RequestResponseFrame{RawFrame: f}
	case core.FrameTypeRequestFNF:
		frame = &FireAndForgetFrame{RawFrame: f}
	case core.FrameTypeRequestStream:
		frame = &RequestStreamFrame{RawFrame: f}
	case core.FrameTypeRequestChannel:
		frame = &RequestChannelFrame{RawFrame: f}
	case core.FrameTypeCancel:
		frame = &CancelFrame{RawFrame: f}
	case core.FrameTypePayload:
		frame = &PayloadFrame{RawFrame: f}
	case core.FrameTypeMetadataPush:
		frame = &MetadataPushFrame{RawFrame: f}
	case core.FrameTypeError:
		frame = &ErrorFrame{RawFrame: f}
	case core.FrameTypeRequestN:
		frame = &RequestNFrame{RawFrame: f}
	case core.FrameTypeLease:
		frame = &LeaseFrame{RawFrame: f}
	case core.FrameTypeResume:
		frame = &ResumeFrame{RawFrame: f}
	case core.FrameTypeResumeOK:
		frame = &ResumeOKFrame{RawFrame: f}
	default:
		err = core.ErrInvalidFrame
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

