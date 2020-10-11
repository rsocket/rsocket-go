package framing

import (
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

// CancelFrame is frame of Cancel.
type CancelFrame struct {
	*bufferedFrame
}

// Validate returns error if frame is invalid.
func (f *CancelFrame) Validate() (err error) {
	// Cancel frame doesn't need any binary body.
	if f.bodyLen() > 0 {
		err = errIncompleteFrame
	}
	return
}

// NewCancelFrame creates cancel frame.
func NewCancelFrame(sid uint32) *CancelFrame {
	bb := common.BorrowByteBuff()
	if err := core.WriteFrameHeader(bb, sid, core.FrameTypeCancel, 0); err != nil {
		common.ReturnByteBuff(bb)
		panic(err)
	}
	return &CancelFrame{
		bufferedFrame: newBufferedFrame(bb),
	}
}
