package framing

import (
	"encoding/binary"
	"fmt"

	"github.com/rsocket/rsocket-go/common"
)

const (
	reqNLen             = 4
	minRequestNFrameLen = reqNLen
)

// FrameRequestN is RequestN frame.
type FrameRequestN struct {
	*BaseFrame
}

// Validate returns error if frame is invalid.
func (p *FrameRequestN) Validate() (err error) {
	if p.body.Len() < minRequestNFrameLen {
		err = errIncompleteFrame
	}
	return
}

func (p *FrameRequestN) String() string {
	return fmt.Sprintf("FrameRequestN{%s,n=%d}", p.header, p.N())
}

// N returns N in RequestN.
func (p *FrameRequestN) N() uint32 {
	return binary.BigEndian.Uint32(p.body.Bytes())
}

// NewFrameRequestN returns a new RequestN frame.
func NewFrameRequestN(sid, n uint32, flags ...FrameFlag) *FrameRequestN {
	fg := newFlags(flags...)
	bf := common.BorrowByteBuffer()
	for i := 0; i < 4; i++ {
		_ = bf.WriteByte(0)
	}
	binary.BigEndian.PutUint32(bf.Bytes(), n)
	return &FrameRequestN{
		&BaseFrame{
			header: NewFrameHeader(sid, FrameTypeRequestN, fg),
			body:   bf,
		},
	}
}
