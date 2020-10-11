package framing

import (
	"encoding/binary"
	"io"

	"github.com/rsocket/rsocket-go/core"
)

// WriteableKeepaliveFrame is writeable Keepalive frame.
type WriteableKeepaliveFrame struct {
	writeableFrame
	pos  [8]byte
	data []byte
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
	t := newWriteableFrame(h)

	return &WriteableKeepaliveFrame{
		writeableFrame: t,
		pos:            b,
		data:           data,
	}
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
