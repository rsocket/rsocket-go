package framing

import (
	"io"

	"github.com/rsocket/rsocket-go/core"
)

// CancelFrame is frame of Cancel.
type CancelFrame struct {
	*baseDefaultFrame
}

// WriteableCancelFrame is writeable frame of Cancel.
type WriteableCancelFrame struct {
	baseWriteableFrame
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

// Validate returns error if frame is invalid.
func (f *CancelFrame) Validate() (err error) {
	// Cancel frame doesn't need any binary body.
	if f.body != nil && f.body.Len() > 0 {
		err = errIncompleteFrame
	}
	return
}

// NewWriteableCancelFrame creates a new WriteableCancelFrame.
func NewWriteableCancelFrame(id uint32) *WriteableCancelFrame {
	h := core.NewFrameHeader(id, core.FrameTypeCancel, 0)
	return &WriteableCancelFrame{
		baseWriteableFrame: newBaseWriteableFrame(h),
	}
}

// NewCancelFrame creates cancel frame.
func NewCancelFrame(sid uint32) *CancelFrame {
	return &CancelFrame{
		newBaseDefaultFrame(core.NewFrameHeader(sid, core.FrameTypeCancel, 0), nil),
	}
}
