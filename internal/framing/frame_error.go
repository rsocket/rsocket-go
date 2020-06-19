package framing

import (
	"encoding/binary"
	"io"
	"strings"

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

type ErrorFrameSupport struct {
	*tinyFrame
	code common.ErrorCode
	data []byte
}

func (e ErrorFrameSupport) Error() string {
	return makeErrorString(e.code, e.data)
}

func (e ErrorFrameSupport) WriteTo(w io.Writer) (n int64, err error) {
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
	return
}

func (e ErrorFrameSupport) Len() int {
	return HeaderLen + 4 + len(e.data)
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
func (p *ErrorFrame) ErrorCode() common.ErrorCode {
	v := binary.BigEndian.Uint32(p.body.Bytes())
	return common.ErrorCode(v)
}

// ErrorData returns error data bytes.
func (p *ErrorFrame) ErrorData() []byte {
	return p.body.Bytes()[errDataOff:]
}

func NewErrorFrameSupport(id uint32, code common.ErrorCode, data []byte) *ErrorFrameSupport {
	h := NewFrameHeader(id, FrameTypeError, 0)
	t := newTinyFrame(h)
	return &ErrorFrameSupport{
		tinyFrame: t,
		code:      code,
		data:      data,
	}
}

// NewErrorFrame returns a new error frame.
func NewErrorFrame(streamID uint32, code common.ErrorCode, data []byte) *ErrorFrame {
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
		NewRawFrame(NewFrameHeader(streamID, FrameTypeError, 0), bf),
	}
}

func makeErrorString(code common.ErrorCode, data []byte) string {
	bu := strings.Builder{}
	bu.WriteString(code.String())
	bu.WriteByte(':')
	bu.WriteByte(' ')
	bu.Write(data)
	return bu.String()
}
