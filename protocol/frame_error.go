package protocol

import (
	"io"
)

type ErrorCode uint32

func (p ErrorCode) String() string {
	switch p {
	case ERR_INVALID_SETUP:
		return "INVALID_SETUP"
	case ERR_UNSUPPORTED_SETUP:
		return "UNSUPPORTED_SETUP"
	case ERR_REJECTED_SETUP:
		return "REJECTED_SETUP"
	case ERR_REJECTED_RESUME:
		return "REJECTED_RESUME"
	case ERR_CONNECTION_ERROR:
		return "CONNECTION_ERROR"
	case ERR_CONNECTION_CLOSE:
		return "CONNECTION_CLOSE"
	case ERR_APPLICATION_ERROR:
		return "APPLICATION_ERROR"
	case ERR_REJECTED:
		return "REJECTED"
	case ERR_CANCELED:
		return "CANCELED"
	case ERR_INVALID:
		return "INVALID"
	case 0x00000000:
		return "RESERVED"
	case 0xFFFFFFFF:
		return "RESERVED"
	default:
		return "UNKNOWN"
	}
}

const (
	ERR_INVALID_SETUP     ErrorCode = 0x00000001
	ERR_UNSUPPORTED_SETUP ErrorCode = 0x00000002
	ERR_REJECTED_SETUP    ErrorCode = 0x00000003
	ERR_REJECTED_RESUME   ErrorCode = 0x00000004
	ERR_CONNECTION_ERROR  ErrorCode = 0x00000101
	ERR_CONNECTION_CLOSE  ErrorCode = 0x00000102
	ERR_APPLICATION_ERROR ErrorCode = 0x00000201
	ERR_REJECTED          ErrorCode = 0x00000202
	ERR_CANCELED          ErrorCode = 0x00000203
	ERR_INVALID           ErrorCode = 0x00000204
)

type FrameError struct {
	Frame
}

func (p FrameError) ErrorCode() ErrorCode {
	return ErrorCode(byteOrder.Uint32(p.Frame[frameHeaderLength : frameHeaderLength+4]))
}

func (p FrameError) ErrorData() []byte {
	return p.Frame[frameHeaderLength+4:]
}

type BaseFrame struct {
	StreamID uint32
	Type     FrameType
	Flags    Flags
}

func (p *BaseFrame) WriteTo(w io.Writer) (n int64, err error) {
	wrote, err := w.Write(p.asBytes())
	n = int64(wrote)
	return
}

func (p *BaseFrame) UnsetFlag(f Flags) {
	p.Flags &= ^f
}

func (p *BaseFrame) SetFlag(f Flags) {
	p.Flags |= f
}

func (p *BaseFrame) asBytes() []byte {
	bs := make([]byte, 6)
	byteOrder.PutUint32(bs, p.StreamID)
	byteOrder.PutUint16(bs[4:], uint16(p.Type)<<10|uint16(p.Flags))
	return bs
}

func NewFrameError(base *BaseFrame, errorCode ErrorCode, errorData []byte) Frame {
	b := make([]byte, 4)
	byteOrder.PutUint32(b, uint32(errorCode))
	bs := make([]byte, 0)
	bs = append(bs, base.asBytes()...)
	bs = append(bs, b...)
	bs = append(bs, errorData...)
	return Frame(bs)
}
