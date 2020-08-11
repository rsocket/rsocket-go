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

// KeepaliveFrame is keepalive frame.
type KeepaliveFrame struct {
	*RawFrame
}

type WriteableKeepaliveFrame struct {
	*tinyFrame
	pos  [8]byte
	data []byte
}

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

func NewWriteableKeepaliveFrame(position uint64, data []byte, respond bool) *WriteableKeepaliveFrame {
	var flag core.FrameFlag
	if respond {
		flag |= core.FlagRespond
	}

	var b [8]byte
	binary.BigEndian.PutUint64(b[:], position)

	h := core.NewFrameHeader(0, core.FrameTypeKeepalive, flag)
	t := newTinyFrame(h)

	return &WriteableKeepaliveFrame{
		tinyFrame: t,
		pos:       b,
		data:      data,
	}
}

// NewKeepaliveFrame returns a new keepalive frame.
func NewKeepaliveFrame(position uint64, data []byte, respond bool) *KeepaliveFrame {
	var fg core.FrameFlag
	if respond {
		fg |= core.FlagRespond
	}
	bf := common.NewByteBuff()
	var b8 [8]byte
	binary.BigEndian.PutUint64(b8[:], position)
	if _, err := bf.Write(b8[:]); err != nil {
		panic(err)
	}
	if len(data) > 0 {
		if _, err := bf.Write(data); err != nil {
			panic(err)
		}
	}
	return &KeepaliveFrame{
		NewRawFrame(core.NewFrameHeader(0, core.FrameTypeKeepalive, fg), bf),
	}
}
