package framing

import (
	"encoding/binary"
	"io"

	"github.com/rsocket/rsocket-go/core"
)

// WriteableResumeOKFrame is writeable ResumeOK frame.
type WriteableResumeOKFrame struct {
	writeableFrame
	pos [8]byte
}

// NewWriteableResumeOKFrame creates a new WriteableResumeOKFrame.
func NewWriteableResumeOKFrame(position uint64) *WriteableResumeOKFrame {
	h := core.NewFrameHeader(0, core.FrameTypeResumeOK, 0)
	t := newWriteableFrame(h)
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], position)
	return &WriteableResumeOKFrame{
		writeableFrame: t,
		pos:            b,
	}
}

// WriteTo writes frame to writer.
func (r WriteableResumeOKFrame) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int64
	wrote, err = r.header.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote
	var v int
	v, err = w.Write(r.pos[:])
	if err != nil {
		return
	}
	n += int64(v)
	return
}

// Len returns length of frame.
func (r WriteableResumeOKFrame) Len() int {
	return core.FrameHeaderLen + 8
}
