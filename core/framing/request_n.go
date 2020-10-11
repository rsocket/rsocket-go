package framing

import (
	"encoding/binary"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

// RequestNFrame is RequestN frame.
type RequestNFrame struct {
	*bufferedFrame
}

// NewRequestNFrame creates a new RequestNFrame.
func NewRequestNFrame(sid, n uint32, fg core.FrameFlag) *RequestNFrame {
	b := common.BorrowByteBuff()

	if err := core.WriteFrameHeader(b, sid, core.FrameTypeRequestN, fg); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}

	if err := binary.Write(b, binary.BigEndian, n); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}

	return &RequestNFrame{
		bufferedFrame: newBufferedFrame(b),
	}
}

// Validate returns error if frame is invalid.
func (r *RequestNFrame) Validate() (err error) {
	if r.bodyLen() != 4 {
		err = errIncompleteFrame
	}
	return
}

// N returns N in RequestN.
func (r *RequestNFrame) N() uint32 {
	return binary.BigEndian.Uint32(r.Body())
}
