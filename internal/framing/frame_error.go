package framing

import (
	"encoding/binary"
	"fmt"

	"github.com/rsocket/rsocket-go/internal/common"
)

const (
	errCodeLen       = 4
	errDataOff       = errCodeLen
	minErrorFrameLen = errCodeLen
)

// FrameError is error frame.
type FrameError struct {
	*BaseFrame
}

// Validate returns error if frame is invalid.
func (p *FrameError) Validate() (err error) {
	if p.Len() < minErrorFrameLen {
		err = errIncompleteFrame
	}
	return
}

func (p *FrameError) Error() string {
	return fmt.Sprintf("%s: %s", p.ErrorCode(), string(p.ErrorData()))
}

// ErrorCode returns error code.
func (p *FrameError) ErrorCode() common.ErrorCode {
	v := binary.BigEndian.Uint32(p.body.Bytes())
	return common.ErrorCode(v)
}

// ErrorData returns error data bytes.
func (p *FrameError) ErrorData() []byte {
	return p.body.Bytes()[errDataOff:]
}

// NewFrameError returns a new error frame.
func NewFrameError(streamID uint32, code common.ErrorCode, data []byte) *FrameError {
	bf := common.BorrowByteBuffer()
	var b4 [4]byte
	binary.BigEndian.PutUint32(b4[:], uint32(code))
	if _, err := bf.Write(b4[:]); err != nil {
		common.ReturnByteBuffer(bf)
		panic(err)
	}
	if _, err := bf.Write(data); err != nil {
		common.ReturnByteBuffer(bf)
		panic(err)
	}
	return &FrameError{
		NewBaseFrame(NewFrameHeader(streamID, FrameTypeError), bf),
	}
}
