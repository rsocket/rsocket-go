package framing

import (
	"encoding/binary"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

const (
	lastRecvPosLen       = 8
	minKeepaliveFrameLen = lastRecvPosLen
)

// KeepaliveFrame is Keepalive frame.
type KeepaliveFrame struct {
	*bufferedFrame
}

// NewKeepaliveFrame creates a new KeepaliveFrame.
func NewKeepaliveFrame(position uint64, data []byte, respond bool) *KeepaliveFrame {
	var fg core.FrameFlag
	if respond {
		fg |= core.FlagRespond
	}
	b := common.BorrowByteBuff()
	if err := core.WriteFrameHeader(b, 0, core.FrameTypeKeepalive, fg); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}
	if err := binary.Write(b, binary.BigEndian, position); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}
	if len(data) > 0 {
		if _, err := b.Write(data); err != nil {
			common.ReturnByteBuff(b)
			panic(err)
		}
	}
	return &KeepaliveFrame{
		bufferedFrame: newBufferedFrame(b),
	}
}

// Validate returns error if frame is invalid.
func (k *KeepaliveFrame) Validate() (err error) {
	if k.bodyLen() < minKeepaliveFrameLen {
		err = errIncompleteFrame
	}
	return
}

// LastReceivedPosition returns last received position.
func (k *KeepaliveFrame) LastReceivedPosition() uint64 {
	return binary.BigEndian.Uint64(k.Body())
}

// Data returns data bytes.
func (k *KeepaliveFrame) Data() []byte {
	return k.Body()[lastRecvPosLen:]
}
