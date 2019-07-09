package framing

import (
	"encoding/binary"
	"fmt"

	"github.com/rsocket/rsocket-go/internal/common"
)

const (
	lastRecvPosLen       = 8
	minKeepaliveFrameLen = lastRecvPosLen
)

// FrameKeepalive is keepalive frame.
type FrameKeepalive struct {
	*BaseFrame
}

func (p *FrameKeepalive) String() string {
	return fmt.Sprintf("FrameKeepalive{%s,lastReceivedPosition=%d,data=%s}", p.header, p.LastReceivedPosition(), string(p.Data()))
}

// Validate returns error if frame is invalid.
func (p *FrameKeepalive) Validate() (err error) {
	if p.body.Len() < minKeepaliveFrameLen {
		err = errIncompleteFrame
	}
	return
}

// LastReceivedPosition returns last received position.
func (p *FrameKeepalive) LastReceivedPosition() uint64 {
	return binary.BigEndian.Uint64(p.body.Bytes())
}

// Data returns data bytes.
func (p *FrameKeepalive) Data() []byte {
	return p.body.Bytes()[lastRecvPosLen:]
}

// NewFrameKeepalive returns a new keepalive frame.
func NewFrameKeepalive(position uint64, data []byte, respond bool) *FrameKeepalive {
	var fg FrameFlag
	if respond {
		fg |= FlagRespond
	}
	bf := common.BorrowByteBuffer()
	var b8 [8]byte
	binary.BigEndian.PutUint64(b8[:], position)
	if _, err := bf.Write(b8[:]); err != nil {
		common.ReturnByteBuffer(bf)
		panic(err)
	}
	if len(data) > 0 {
		if _, err := bf.Write(data); err != nil {
			common.ReturnByteBuffer(bf)
			panic(err)
		}
	}
	return &FrameKeepalive{
		NewBaseFrame(NewFrameHeader(0, FrameTypeKeepalive, fg), bf),
	}
}
