package framing

import (
	"encoding/binary"
	"io"

	"github.com/rsocket/rsocket-go/core"
)

// WriteableRequestStreamFrame is writeable RequestStream frame.
type WriteableRequestStreamFrame struct {
	writeableFrame
	n        [4]byte
	metadata []byte
	data     []byte
}

// NewWriteableRequestStreamFrame creates a new WriteableRequestStreamFrame.
func NewWriteableRequestStreamFrame(id uint32, n uint32, data, metadata []byte, flag core.FrameFlag) *WriteableRequestStreamFrame {
	if len(metadata) > 0 {
		flag |= core.FlagMetadata
	}
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], n)
	h := core.NewFrameHeader(id, core.FrameTypeRequestStream, flag)
	t := newWriteableFrame(h)
	return &WriteableRequestStreamFrame{
		writeableFrame: t,
		n:              b,
		metadata:       metadata,
		data:           data,
	}
}

// WriteTo writes frame to writer.
func (r WriteableRequestStreamFrame) WriteTo(w io.Writer) (n int64, err error) {
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
func (r WriteableRequestStreamFrame) Len() int {
	return 4 + CalcPayloadFrameSize(r.data, r.metadata)
}
