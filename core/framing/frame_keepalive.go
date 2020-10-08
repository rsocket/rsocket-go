package framing

import (
	"encoding/binary"
	"io"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

const (
	lastRecvPosLen       = 8
	minKeepaliveFrameLen = lastRecvPosLen
)

// KeepaliveFrame is Keepalive frame.
type KeepaliveFrame struct {
	*baseDefaultFrame
}

// WriteableKeepaliveFrame is writeable Keepalive frame.
type WriteableKeepaliveFrame struct {
	baseWriteableFrame
	pos  [8]byte
	data []byte
}

// WriteTo writes frame to writer.
func (k WriteableKeepaliveFrame) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int64
	wrote, err = k.header.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote

	var v int
	v, err = w.Write(k.pos[:])
	if err != nil {
		return
	}
	n += int64(v)

	v, err = w.Write(k.data)
	if err != nil {
		return
	}
	n += int64(v)

	return
}

// Len returns length of frame.
func (k WriteableKeepaliveFrame) Len() int {
	return core.FrameHeaderLen + 8 + len(k.data)
}

// Validate returns error if frame is invalid.
func (k *KeepaliveFrame) Validate() (err error) {
	if k.body.Len() < minKeepaliveFrameLen {
		err = errIncompleteFrame
	}
	return
}

// LastReceivedPosition returns last received position.
func (k *KeepaliveFrame) LastReceivedPosition() uint64 {
	return binary.BigEndian.Uint64(k.body.Bytes())
}

// Data returns data bytes.
func (k *KeepaliveFrame) Data() []byte {
	return k.body.Bytes()[lastRecvPosLen:]
}

// NewWriteableKeepaliveFrame creates a new WriteableKeepaliveFrame.
func NewWriteableKeepaliveFrame(position uint64, data []byte, respond bool) *WriteableKeepaliveFrame {
	var flag core.FrameFlag
	if respond {
		flag |= core.FlagRespond
	}

	var b [8]byte
	binary.BigEndian.PutUint64(b[:], position)

	h := core.NewFrameHeader(0, core.FrameTypeKeepalive, flag)
	t := newBaseWriteableFrame(h)

	return &WriteableKeepaliveFrame{
		baseWriteableFrame: t,
		pos:                b,
		data:               data,
	}
}

// NewKeepaliveFrame creates a new KeepaliveFrame.
func NewKeepaliveFrame(position uint64, data []byte, respond bool) *KeepaliveFrame {
	var fg core.FrameFlag
	if respond {
		fg |= core.FlagRespond
	}
	b := common.BorrowByteBuff()
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
		newBaseDefaultFrame(core.NewFrameHeader(0, core.FrameTypeKeepalive, fg), b),
	}
}
