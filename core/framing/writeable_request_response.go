package framing

import (
	"io"

	"github.com/rsocket/rsocket-go/core"
)

// WriteableRequestResponseFrame is writeable RequestResponse frame.
type WriteableRequestResponseFrame struct {
	writeableFrame
	metadata []byte
	data     []byte
}

// NewWriteableRequestResponseFrame returns a new WriteableRequestResponseFrame.
func NewWriteableRequestResponseFrame(id uint32, data, metadata []byte, fg core.FrameFlag) *WriteableRequestResponseFrame {
	if len(metadata) > 0 {
		fg |= core.FlagMetadata
	}
	return &WriteableRequestResponseFrame{
		writeableFrame: newWriteableFrame(core.NewFrameHeader(id, core.FrameTypeRequestResponse, fg)),
		metadata:       metadata,
		data:           data,
	}
}

// WriteTo writes frame to writer.
func (r WriteableRequestResponseFrame) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int64
	wrote, err = r.header.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote
	wrote, err = writePayload(w, r.data, r.metadata)
	if err == nil {
		n += wrote
	}
	return
}

// Len returns length of frame.
func (r WriteableRequestResponseFrame) Len() int {
	return CalcPayloadFrameSize(r.data, r.metadata)
}
