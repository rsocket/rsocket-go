package framing

import (
	"encoding/binary"
	"io"

	"github.com/rsocket/rsocket-go/core"
)

// WriteableRequestNFrame is writeable RequestN frame.
type WriteableRequestNFrame struct {
	writeableFrame
	n [4]byte
}

// NewWriteableRequestNFrame creates a new WriteableRequestNFrame.
func NewWriteableRequestNFrame(id uint32, n uint32, fg core.FrameFlag) *WriteableRequestNFrame {
	var b4 [4]byte
	binary.BigEndian.PutUint32(b4[:], n)
	return &WriteableRequestNFrame{
		writeableFrame: newWriteableFrame(core.NewFrameHeader(id, core.FrameTypeRequestN, fg)),
		n:              b4,
	}
}

// WriteTo writes frame to writer.
func (r WriteableRequestNFrame) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int64
	wrote, err = r.header.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote
	v, err := w.Write(r.n[:])
	if err == nil {
		n += int64(v)
	}
	return
}

// Len returns length of frame.
func (r WriteableRequestNFrame) Len() int {
	return core.FrameHeaderLen + 4
}
