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
	*RawFrame
}

type WriteableErrorFrame struct {
	*tinyFrame
	code core.ErrorCode
	data []byte
}

func (e WriteableErrorFrame) Error() string {
	return makeErrorString(e.code, e.data)
}

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

func (e WriteableErrorFrame) Len() int {
	return core.FrameHeaderLen + 4 + len(e.data)
}

// Validate returns error if frame is invalid.
func (p *ErrorFrame) Validate() (err error) {
	if p.body.Len() < minErrorFrameLen {
		err = errIncompleteFrame
	}
	return
}

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

func NewWriteableErrorFrame(id uint32, code core.ErrorCode, data []byte) *WriteableErrorFrame {
	h := core.NewFrameHeader(id, core.FrameTypeError, 0)
	t := newTinyFrame(h)
	return &WriteableErrorFrame{
		tinyFrame: t,
		code:      code,
		data:      data,
	}
}

// NewErrorFrame returns a new error frame.
func NewErrorFrame(streamID uint32, code core.ErrorCode, data []byte) *ErrorFrame {
	bf := common.NewByteBuff()
	var b4 [4]byte
	binary.BigEndian.PutUint32(b4[:], uint32(code))
	if _, err := bf.Write(b4[:]); err != nil {
		panic(err)
	}
	if _, err := bf.Write(data); err != nil {
		panic(err)
	}
	return &ErrorFrame{
		NewRawFrame(core.NewFrameHeader(streamID, core.FrameTypeError, 0), bf),
	}
}

func makeErrorString(code core.ErrorCode, data []byte) string {
	bu := strings.Builder{}
	bu.WriteString(code.String())
	bu.WriteByte(':')
	bu.WriteByte(' ')
	bu.Write(data)
	return bu.String()
}
