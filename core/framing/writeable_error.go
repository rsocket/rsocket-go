package framing

import (
	"encoding/binary"
	"io"

	"github.com/rsocket/rsocket-go/core"
)

// WriteableErrorFrame is writeable error frame.
type WriteableErrorFrame struct {
	writeableFrame
	frozenError
}

// NewWriteableErrorFrame creates WriteableErrorFrame.
func NewWriteableErrorFrame(id uint32, code core.ErrorCode, data []byte) *WriteableErrorFrame {
	h := core.NewFrameHeader(id, core.FrameTypeError, 0)
	t := newWriteableFrame(h)
	return &WriteableErrorFrame{
		writeableFrame: t,
		frozenError: frozenError{
			code: code,
			data: data,
		},
	}
}

// WriteTo writes frame to writer.
func (e WriteableErrorFrame) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int64
	wrote, err = e.header.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote

	err = binary.Write(w, binary.BigEndian, uint32(e.code))
	if err != nil {
		return
	}
	n += 4

	l, err := w.Write(e.data)
	if err != nil {
		return
	}
	n += int64(l)
	return
}

// Len returns length of frame.
func (e WriteableErrorFrame) Len() int {
	return core.FrameHeaderLen + 4 + len(e.data)
}
