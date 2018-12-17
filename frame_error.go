package rsocket

import (
	"encoding/binary"
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
	*Header
	code ErrorCode
	data []byte
}

func (p *FrameError) Size() int {
	size := headerLen + 4
	if p.data != nil {
		size += len(p.data)
	}
	return size
}

func (p *FrameError) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int
	wrote, err = w.Write(p.Header.Bytes())
	n += int64(wrote)
	if err != nil {
		return
	}
	b4 := make([]byte, 4)
	binary.BigEndian.PutUint32(b4, uint32(p.ErrorCode()))
	wrote, err = w.Write(b4)
	n += int64(wrote)
	if err != nil {
		return
	}
	if p.data == nil {
		return
	}
	wrote, err = w.Write(p.data)
	n += int64(wrote)
	return
}

func (p *FrameError) ErrorCode() ErrorCode {
	return p.code
}

func (p *FrameError) ErrorData() []byte {
	return p.data
}

func (p *FrameError) Parse(h *Header, bs []byte) error {
	p.Header = h
	p.code = ErrorCode(binary.BigEndian.Uint32(bs[headerLen : headerLen+4]))
	data := bs[headerLen+4:]
	if data == nil {
		p.data = nil
	} else {
		p.data = make([]byte, len(data))
		copy(p.data, data)
	}
	return nil
}

func mkError(sid uint32, code ErrorCode, data []byte) *FrameError {
	return &FrameError{
		Header: mkHeader(sid, ERROR),
		code:   code,
		data:   data,
	}
}
