package framing

import (
	"encoding/binary"
	"fmt"

	"github.com/rsocket/rsocket-go/internal/common"
)

type FrameResumeOK struct {
	*BaseFrame
}

func (p *FrameResumeOK) String() string {
	return fmt.Sprintf("FrameResumeOK{%s,lastReceivedClientPosition=%d}", p.header, p.LastReceivedClientPosition())
}

func (p *FrameResumeOK) Validate() (err error) {
	return
}

func (p *FrameResumeOK) LastReceivedClientPosition() uint64 {
	raw := p.body.Bytes()
	return binary.BigEndian.Uint64(raw)
}

func NewResumeOK(position uint64) *FrameResumeOK {
	var b8 [8]byte
	binary.BigEndian.PutUint64(b8[:], position)
	bf := common.BorrowByteBuffer()
	_, err := bf.Write(b8[:])
	if err != nil {
		common.ReturnByteBuffer(bf)
		panic(err)
	}
	return &FrameResumeOK{
		&BaseFrame{
			header: NewFrameHeader(0, FrameTypeResumeOK),
			body:   bf,
		},
	}
}
