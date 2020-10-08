package framing

import (
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"

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

// FromRawFrame creates a frame from a baseDefaultFrame.
func FromRawFrame(f *baseDefaultFrame) (frame core.BufferedFrame, err error) {
	switch f.header.Type() {
	case core.FrameTypeSetup:
		frame = &SetupFrame{baseDefaultFrame: f}
	case core.FrameTypeKeepalive:
		frame = &KeepaliveFrame{baseDefaultFrame: f}
	case core.FrameTypeRequestResponse:
		frame = &RequestResponseFrame{baseDefaultFrame: f}
	case core.FrameTypeRequestFNF:
		frame = &FireAndForgetFrame{baseDefaultFrame: f}
	case core.FrameTypeRequestStream:
		frame = &RequestStreamFrame{baseDefaultFrame: f}
	case core.FrameTypeRequestChannel:
		frame = &RequestChannelFrame{baseDefaultFrame: f}
	case core.FrameTypeCancel:
		frame = &CancelFrame{baseDefaultFrame: f}
	case core.FrameTypePayload:
		frame = &PayloadFrame{baseDefaultFrame: f}
	case core.FrameTypeMetadataPush:
		frame = &MetadataPushFrame{baseDefaultFrame: f}
	case core.FrameTypeError:
		frame = &ErrorFrame{baseDefaultFrame: f}
	case core.FrameTypeRequestN:
		frame = &RequestNFrame{baseDefaultFrame: f}
	case core.FrameTypeLease:
		frame = &LeaseFrame{baseDefaultFrame: f}
	case core.FrameTypeResume:
		frame = &ResumeFrame{baseDefaultFrame: f}
	case core.FrameTypeResumeOK:
		frame = &ResumeOKFrame{baseDefaultFrame: f}
	default:
		err = core.ErrInvalidFrame
	}
	return
}

// PrintFrame prints frame in bytes dump.
func PrintFrame(f core.Frame) string {
	var initN, reqN uint32
	var metadata, data []byte

	switch it := f.(type) {
	case *PayloadFrame:
		metadata, _ = it.Metadata()
		data = it.Data()
	case *WriteablePayloadFrame:
		metadata, data = it.metadata, it.data
	case *MetadataPushFrame:
		metadata, _ = it.Metadata()
	case *FireAndForgetFrame:
		metadata, _ = it.Metadata()
		data = it.Data()
	case *RequestResponseFrame:
		metadata, _ = it.Metadata()
		data = it.Data()
	case *RequestStreamFrame:
		metadata, _ = it.Metadata()
		data = it.Data()
		initN = it.InitialRequestN()
	case *RequestChannelFrame:
		metadata, _ = it.Metadata()
		data = it.Data()
		initN = it.InitialRequestN()
	case *SetupFrame:
		metadata, _ = it.Metadata()
		data = it.Data()
	case *RequestNFrame:
		reqN = it.N()
	case *WriteableMetadataPushFrame:
		metadata = it.metadata
	case *WriteableFireAndForgetFrame:
		metadata, data = it.metadata, it.data
	case *WriteableRequestResponseFrame:
		metadata, data = it.metadata, it.data
	case *WriteableRequestStreamFrame:
		metadata, data = it.metadata, it.data
		reqN = binary.BigEndian.Uint32(it.n[:])
	case *WriteableRequestChannelFrame:
		metadata, data = it.metadata, it.data
		reqN = binary.BigEndian.Uint32(it.n[:])
	case *WriteableSetupFrame:
		metadata, data = it.metadata, it.data
	case *WriteableRequestNFrame:
		reqN = binary.BigEndian.Uint32(it.n[:])
	}

	b := &strings.Builder{}
	b.WriteString("\nFrame => Stream ID: ")
	h := f.Header()
	b.WriteString(strconv.Itoa(int(h.StreamID())))
	b.WriteString(" Type: ")
	b.WriteString(h.Type().String())
	b.WriteString(" Flags: 0b")
	_, _ = fmt.Fprintf(b, "%010b", h.Flag())
	b.WriteString(" Length: ")
	b.WriteString(strconv.Itoa(f.Len()))
	if initN > 0 {
		b.WriteString(" InitialRequestN: ")
		_, _ = fmt.Fprintf(b, "%d", initN)
	}

	if reqN > 0 {
		b.WriteString(" RequestN: ")
		_, _ = fmt.Fprintf(b, "%d", reqN)
	}

	if metadata != nil {
		b.WriteString("\nMetadata:\n")
		common.AppendPrettyHexDump(b, metadata)
	}

	if data != nil {
		b.WriteString("\nData:\n")
		common.AppendPrettyHexDump(b, data)
	}
	return b.String()
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
