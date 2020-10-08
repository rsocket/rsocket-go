package framing

import (
	"encoding/binary"
	"io"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

// RequestNFrame is RequestN frame.
type RequestNFrame struct {
	*baseDefaultFrame
}

// WriteableRequestNFrame is writeable RequestN frame.
type WriteableRequestNFrame struct {
	baseWriteableFrame
	n [4]byte
}

// Validate returns error if frame is invalid.
func (r *RequestNFrame) Validate() (err error) {
	if r.body.Len() != 4 {
		err = errIncompleteFrame
	}
	return
}

// N returns N in RequestN.
func (r *RequestNFrame) N() uint32 {
	return binary.BigEndian.Uint32(r.body.Bytes())
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

// NewWriteableRequestNFrame creates a new WriteableRequestNFrame.
func NewWriteableRequestNFrame(id uint32, n uint32, fg core.FrameFlag) *WriteableRequestNFrame {
	var b4 [4]byte
	binary.BigEndian.PutUint32(b4[:], n)
	return &WriteableRequestNFrame{
		baseWriteableFrame: newBaseWriteableFrame(core.NewFrameHeader(id, core.FrameTypeRequestN, fg)),
		n:                  b4,
	}
}

// NewRequestNFrame creates a new RequestNFrame.
func NewRequestNFrame(sid, n uint32, fg core.FrameFlag) *RequestNFrame {
	b := common.BorrowByteBuff()
	if err := binary.Write(b, binary.BigEndian, n); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}
	return &RequestNFrame{
		newBaseDefaultFrame(core.NewFrameHeader(sid, core.FrameTypeRequestN, fg), b),
	}
}
