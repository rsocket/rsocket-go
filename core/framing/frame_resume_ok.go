package framing

import (
	"encoding/binary"
	"io"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

// ResumeOKFrame represents a frame of ResumeOK.
type ResumeOKFrame struct {
	*RawFrame
}

type WriteableResumeOKFrame struct {
	*tinyFrame
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

func (r WriteableResumeOKFrame) Len() int {
	return core.FrameHeaderLen + 8
}

func NewWriteableResumeOKFrame(position uint64) *WriteableResumeOKFrame {
	h := core.NewFrameHeader(0, core.FrameTypeResumeOK, 0)
	t := newTinyFrame(h)
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], position)
	return &WriteableResumeOKFrame{
		tinyFrame: t,
		pos:       b,
	}
}

// NewResumeOKFrame creates a new frame of ResumeOK.
func NewResumeOKFrame(position uint64) *ResumeOKFrame {
	var b8 [8]byte
	binary.BigEndian.PutUint64(b8[:], position)
	bf := common.NewByteBuff()
	_, err := bf.Write(b8[:])
	if err != nil {
		panic(err)
	}
	return &ResumeOKFrame{
		NewRawFrame(core.NewFrameHeader(0, core.FrameTypeResumeOK, 0), bf),
	}
}