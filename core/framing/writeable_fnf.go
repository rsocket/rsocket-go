package framing

import (
	"io"

	"github.com/rsocket/rsocket-go/core"
)

// WriteableFireAndForgetFrame is writeable FireAndForget frame.
type WriteableFireAndForgetFrame struct {
	writeableFrame
	metadata []byte
	data     []byte
}

// NewWriteableFireAndForgetFrame creates a new WriteableFireAndForgetFrame.
func NewWriteableFireAndForgetFrame(sid uint32, data, metadata []byte, flag core.FrameFlag) *WriteableFireAndForgetFrame {
	if len(metadata) > 0 {
		flag |= core.FlagMetadata
	}
	h := core.NewFrameHeader(sid, core.FrameTypeRequestFNF, flag)
	t := newWriteableFrame(h)
	return &WriteableFireAndForgetFrame{
		writeableFrame: t,
		metadata:       metadata,
		data:           data,
	}
}

// WriteTo writes frame to writer.
func (f WriteableFireAndForgetFrame) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int64
	wrote, err = f.header.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote

	wrote, err = writePayload(w, f.data, f.metadata)
	if err != nil {
		return
	}
	n += wrote
	return
}

// Len returns length of frame.
func (f WriteableFireAndForgetFrame) Len() int {
	return CalcPayloadFrameSize(f.data, f.metadata)
}
