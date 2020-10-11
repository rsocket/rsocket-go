package framing

import (
	"encoding/binary"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

// ResumeOKFrame is ResumeOK frame.
type ResumeOKFrame struct {
	*bufferedFrame
}

// NewResumeOKFrame creates a new ResumeOKFrame.
func NewResumeOKFrame(position uint64) *ResumeOKFrame {
	b := common.BorrowByteBuff()

	if err := core.WriteFrameHeader(b, 0, core.FrameTypeResumeOK, 0); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}

	if err := binary.Write(b, binary.BigEndian, position); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}

	return &ResumeOKFrame{
		bufferedFrame: newBufferedFrame(b),
	}
}

// Validate validate current frame.
func (r *ResumeOKFrame) Validate() (err error) {
	// Length of frame body should be 8
	if r.bodyLen() != 8 {
		err = errIncompleteFrame
	}
	return
}

// LastReceivedClientPosition returns last received client position.
func (r *ResumeOKFrame) LastReceivedClientPosition() uint64 {
	raw := r.Body()
	return binary.BigEndian.Uint64(raw)
}
