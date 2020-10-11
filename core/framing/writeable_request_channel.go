package framing

import (
	"encoding/binary"
	"io"

	"github.com/rsocket/rsocket-go/core"
)

// WriteableRequestChannelFrame is writeable RequestChannel frame.
type WriteableRequestChannelFrame struct {
	writeableFrame
	n        [4]byte
	metadata []byte
	data     []byte
}

// NewWriteableRequestChannelFrame creates a new WriteableRequestChannelFrame.
func NewWriteableRequestChannelFrame(sid uint32, n uint32, data, metadata []byte, flag core.FrameFlag) *WriteableRequestChannelFrame {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], n)
	if len(metadata) > 0 {
		flag |= core.FlagMetadata
	}
	h := core.NewFrameHeader(sid, core.FrameTypeRequestChannel, flag)
	t := newWriteableFrame(h)
	return &WriteableRequestChannelFrame{
		writeableFrame: t,
		n:              b,
		metadata:       metadata,
		data:           data,
	}
}

// WriteTo writes frame to writer.
func (r WriteableRequestChannelFrame) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int64
	wrote, err = r.header.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote

	var v int
	v, err = w.Write(r.n[:])
	if err != nil {
		return
	}
	n += int64(v)

	wrote, err = writePayload(w, r.data, r.metadata)
	if err != nil {
		return
	}
	n += wrote

	return
}

// Len returns length of frame.
func (r WriteableRequestChannelFrame) Len() int {
	return CalcPayloadFrameSize(r.data, r.metadata) + 4
}
