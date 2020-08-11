package framing

import (
	"encoding/binary"
	"io"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

// RequestNFrame is RequestN frame.
type RequestNFrame struct {
	*RawFrame
}

type WriteableRequestNFrame struct {
	*tinyFrame
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

func (r WriteableRequestNFrame) Len() int {
	return core.FrameHeaderLen + 4
}

func NewWriteableRequestNFrame(id uint32, n uint32, fg core.FrameFlag) *WriteableRequestNFrame {
	var b4 [4]byte
	binary.BigEndian.PutUint32(b4[:], n)
	return &WriteableRequestNFrame{
		tinyFrame: newTinyFrame(core.NewFrameHeader(id, core.FrameTypeRequestN, fg)),
		n:         b4,
	}
}

// NewRequestNFrame returns a new RequestN frame.
func NewRequestNFrame(sid, n uint32, fg core.FrameFlag) *RequestNFrame {
	bf := common.NewByteBuff()
	var b4 [4]byte
	binary.BigEndian.PutUint32(b4[:], n)
	if _, err := bf.Write(b4[:]); err != nil {
		panic(err)
	}
	return &RequestNFrame{
		NewRawFrame(core.NewFrameHeader(sid, core.FrameTypeRequestN, fg), bf),
	}
}
