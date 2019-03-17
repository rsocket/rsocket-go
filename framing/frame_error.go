package framing

import (
	"encoding/binary"
	"fmt"
	"github.com/rsocket/rsocket-go/common"
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
	b4 := common.BorrowByteBuffer()
	defer common.ReturnByteBuffer(b4)
	for range [4]struct{}{} {
		_ = b4.WriteByte(0)
	}
	binary.BigEndian.PutUint32(b4.Bytes(), uint32(code))
	_, _ = b4.WriteTo(bf)
	_, _ = bf.Write(data)
	return &FrameError{
		&BaseFrame{
			header: NewFrameHeader(streamID, FrameTypeError),
			body:   bf,
		},
	}
}
