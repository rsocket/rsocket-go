package framing

import (
	"encoding/binary"
	"io"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

// ResumeOKFrame is ResumeOK frame.
type ResumeOKFrame struct {
	*baseDefaultFrame
}

// WriteableResumeOKFrame is writeable ResumeOK frame.
type WriteableResumeOKFrame struct {
	baseWriteableFrame
	pos [8]byte
}

// Validate validate current frame.
func (r *ResumeOKFrame) Validate() (err error) {
	// Length of frame body should be 8
	if r.body.Len() != 8 {
		err = errIncompleteFrame
	}
	return
}

// LastReceivedClientPosition returns last received client position.
func (r *ResumeOKFrame) LastReceivedClientPosition() uint64 {
	raw := r.body.Bytes()
	return binary.BigEndian.Uint64(raw)
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

// NewWriteableResumeOKFrame creates a new WriteableResumeOKFrame.
func NewWriteableResumeOKFrame(position uint64) *WriteableResumeOKFrame {
	h := core.NewFrameHeader(0, core.FrameTypeResumeOK, 0)
	t := newBaseWriteableFrame(h)
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], position)
	return &WriteableResumeOKFrame{
		baseWriteableFrame: t,
		pos:                b,
	}
}

// NewResumeOKFrame creates a new ResumeOKFrame.
func NewResumeOKFrame(position uint64) *ResumeOKFrame {
	b := common.BorrowByteBuff()
	if err := binary.Write(b, binary.BigEndian, position); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}
	return &ResumeOKFrame{
		newBaseDefaultFrame(core.NewFrameHeader(0, core.FrameTypeResumeOK, 0), b),
	}
}
