package framing

import (
	"encoding/binary"
	"fmt"
	"github.com/rsocket/rsocket-go/common"
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
	return fmt.Sprintf("FrameKeepalive{%s,lastReceivedPosition=%d,data=%s}", p.Header(), p.LastReceivedPosition(), string(p.Data()))
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
	for range [8]struct{}{} {
		_ = bf.WriteByte(0)
	}
	binary.BigEndian.PutUint64(bf.Bytes(), position)
	if len(data) > 0 {
		_, _ = bf.Write(data)
	}
	return &FrameKeepalive{
		&BaseFrame{
			header: NewFrameHeader(0, FrameTypeKeepalive, fg),
			body:   bf,
		},
	}
}
