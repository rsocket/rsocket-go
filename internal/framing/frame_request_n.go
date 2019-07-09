package framing

import (
	"encoding/binary"
	"fmt"

	"github.com/rsocket/rsocket-go/internal/common"
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

	var b4 [4]byte
	binary.BigEndian.PutUint32(b4[:], n)
	if _, err := bf.Write(b4[:]); err != nil {
		common.ReturnByteBuffer(bf)
		panic(err)
	}
	return &FrameRequestN{
		NewBaseFrame(NewFrameHeader(sid, FrameTypeRequestN, fg), bf),
	}
}
