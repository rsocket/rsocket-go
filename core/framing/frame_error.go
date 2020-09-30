package framing

import (
	"encoding/binary"
	"io"
	"strings"

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
	*baseDefaultFrame
}

type frozenError struct {
	code core.ErrorCode
	data []byte
}

// WriteableErrorFrame is writeable error frame.
type WriteableErrorFrame struct {
	baseWriteableFrame
	frozenError
}

// NewWriteableErrorFrame creates WriteableErrorFrame.
func NewWriteableErrorFrame(id uint32, code core.ErrorCode, data []byte) *WriteableErrorFrame {
	h := core.NewFrameHeader(id, core.FrameTypeError, 0)
	t := newBaseWriteableFrame(h)
	return &WriteableErrorFrame{
		baseWriteableFrame: t,
		frozenError: frozenError{
			code: code,
			data: data,
		},
	}
}

// NewErrorFrame returns a new error frame.
func NewErrorFrame(streamID uint32, code core.ErrorCode, data []byte) *ErrorFrame {
	b := common.BorrowByteBuff()
	if err := binary.Write(b, binary.BigEndian, uint32(code)); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}
	if _, err := b.Write(data); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}
	return &ErrorFrame{
		newBaseDefaultFrame(core.NewFrameHeader(streamID, core.FrameTypeError, 0), b),
	}
}

func (c frozenError) Error() string {
	return makeErrorString(c.ErrorCode(), c.ErrorData())
}

func (c frozenError) ErrorCode() core.ErrorCode {
	return c.code
}

func (c frozenError) ErrorData() []byte {
	return c.data
}

// WriteTo writes frame to writer.
func (e WriteableErrorFrame) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int64
	wrote, err = e.header.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote

	err = binary.Write(w, binary.BigEndian, uint32(e.code))
	if err != nil {
		return
	}
	n += 4

	l, err := w.Write(e.data)
	if err != nil {
		return
	}
	n += int64(l)
	return
}

// Len returns length of frame.
func (e WriteableErrorFrame) Len() int {
	return core.FrameHeaderLen + 4 + len(e.data)
}

func (p *ErrorFrame) ToError() error {
	return frozenError{
		code: p.ErrorCode(),
		data: common.CloneBytes(p.ErrorData()),
	}
}

// Validate returns error if frame is invalid.
func (p *ErrorFrame) Validate() (err error) {
	if p.body.Len() < minErrorFrameLen {
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
	v := binary.BigEndian.Uint32(p.body.Bytes())
	return core.ErrorCode(v)
}

// ErrorData returns error data bytes.
func (p *ErrorFrame) ErrorData() []byte {
	return p.body.Bytes()[errDataOff:]
}

func makeErrorString(code core.ErrorCode, data []byte) string {
	bu := strings.Builder{}
	bu.WriteString(code.String())
	bu.WriteByte(':')
	bu.WriteByte(' ')
	bu.Write(data)
	return bu.String()
}
