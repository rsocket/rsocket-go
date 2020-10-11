package framing

import (
	"errors"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

var errIncompleteFrame = errors.New("incomplete frame")

// FromBytes creates frame from a byte slice.
func FromBytes(b []byte) (f core.BufferedFrame, err error) {
	if len(b) < core.FrameHeaderLen {
		err = errIncompleteFrame
		return
	}
	bb := common.BorrowByteBuff()
	_, err = bb.Write(b)
	if err != nil {
		common.ReturnByteBuff(bb)
		return
	}
	f, err = convert(newBufferedFrame(bb))
	if err != nil {
		common.ReturnByteBuff(bb)
	}
	return
}

// CalcPayloadFrameSize returns payload frame size.
func CalcPayloadFrameSize(data, metadata []byte) int {
	size := core.FrameHeaderLen + len(data)
	if n := len(metadata); n > 0 {
		size += 3 + n
	}
	return size
}
