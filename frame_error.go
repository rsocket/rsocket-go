package rsocket

import (
	"encoding/binary"
	"fmt"
)

type frameError struct {
	*baseFrame
}

func (p *frameError) Error() string {
	return fmt.Sprintf("%s: %s", p.ErrorCode(), string(p.ErrorData()))
}

func (p *frameError) ErrorCode() ErrorCode {
	v := binary.BigEndian.Uint32(p.body.Bytes())
	return ErrorCode(v)
}

func (p *frameError) ErrorData() []byte {
	return p.body.Bytes()[4:]
}

func createError(streamID uint32, code ErrorCode, data []byte) *frameError {
	bf := borrowByteBuffer()
	b4 := borrowByteBuffer()
	defer returnByteBuffer(b4)
	for i := 0; i < 4; i++ {
		_ = b4.WriteByte(0)
	}
	binary.BigEndian.PutUint32(b4.Bytes(), uint32(code))
	_, _ = b4.WriteTo(bf)
	_, _ = bf.Write(data)
	return &frameError{
		&baseFrame{
			header: createHeader(streamID, tError),
			body:   bf,
		},
	}
}
