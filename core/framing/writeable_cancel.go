package framing

import (
	"io"

	"github.com/rsocket/rsocket-go/core"
)

// WriteableCancelFrame is writeable frame of Cancel.
type WriteableCancelFrame struct {
	writeableFrame
}

// NewWriteableCancelFrame creates a new WriteableCancelFrame.
func NewWriteableCancelFrame(id uint32) *WriteableCancelFrame {
	h := core.NewFrameHeader(id, core.FrameTypeCancel, 0)
	return &WriteableCancelFrame{
		writeableFrame: newWriteableFrame(h),
	}
}

// WriteTo writes current frame to given writer.
func (c WriteableCancelFrame) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int64
	wrote, err = c.header.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote
	return
}

// Len returns length of frame.
func (c WriteableCancelFrame) Len() int {
	return core.FrameHeaderLen
}
