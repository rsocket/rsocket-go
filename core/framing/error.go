package framing

import (
	"encoding/binary"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

const (
	errCodeLen       = 4
	errDataOff       = errCodeLen
	minErrorFrameLen = errCodeLen
)

// ErrorFrame is error frame.
type ErrorFrame struct {
	*bufferedFrame
}

// NewErrorFrame returns a new error frame.
func NewErrorFrame(sid uint32, code core.ErrorCode, data []byte) *ErrorFrame {
	b := common.BorrowByteBuff()

	if err := core.WriteFrameHeader(b, sid, core.FrameTypeError, 0); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}

	if err := binary.Write(b, binary.BigEndian, uint32(code)); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}

	if _, err := b.Write(data); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}

	return &ErrorFrame{
		bufferedFrame: newBufferedFrame(b),
	}
}

func (p *ErrorFrame) ToError() error {
	return frozenError{
		code: p.ErrorCode(),
		data: common.CloneBytes(p.ErrorData()),
	}
}

// Validate returns error if frame is invalid.
func (p *ErrorFrame) Validate() (err error) {
	if p.bodyLen() < minErrorFrameLen {
		err = errIncompleteFrame
	}
	return
}

// Error returns error string.
func (p *ErrorFrame) Error() string {
	return makeErrorString(p.ErrorCode(), p.ErrorData())
}

// ErrorCode returns error code.
func (p *ErrorFrame) ErrorCode() core.ErrorCode {
	v := binary.BigEndian.Uint32(p.Body())
	return core.ErrorCode(v)
}

// ErrorData returns error data bytes.
func (p *ErrorFrame) ErrorData() []byte {
	return p.Body()[errDataOff:]
}
