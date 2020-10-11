package framing

import (
	"encoding/binary"
	"io"

	"github.com/rsocket/rsocket-go/core"
)

// WriteableResumeFrame is writeable Resume frame.
type WriteableResumeFrame struct {
	writeableFrame
	version  core.Version
	token    []byte
	posFirst [8]byte
	posLast  [8]byte
}

// NewWriteableResumeFrame creates a new WriteableResumeFrame.
func NewWriteableResumeFrame(
	version core.Version,
	token []byte,
	firstAvailableClientPosition uint64,
	lastReceivedServerPosition uint64,
) *WriteableResumeFrame {
	h := core.NewFrameHeader(0, core.FrameTypeResume, 0)
	t := newWriteableFrame(h)
	var a, b [8]byte
	binary.BigEndian.PutUint64(a[:], firstAvailableClientPosition)
	binary.BigEndian.PutUint64(b[:], lastReceivedServerPosition)

	return &WriteableResumeFrame{
		writeableFrame: t,
		version:        version,
		token:          token,
		posFirst:       a,
		posLast:        b,
	}
}

// WriteTo writes frame to writer.
func (r WriteableResumeFrame) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int64
	wrote, err = r.header.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote

	var v int

	v, err = w.Write(r.version.Bytes())
	if err != nil {
		return
	}
	n += int64(v)

	lenToken := uint16(len(r.token))
	err = binary.Write(w, binary.BigEndian, lenToken)
	if err != nil {
		return
	}
	n += 2

	v, err = w.Write(r.token)
	if err != nil {
		return
	}
	n += int64(v)

	v, err = w.Write(r.posLast[:])
	if err != nil {
		return
	}
	n += int64(v)

	v, err = w.Write(r.posFirst[:])
	if err != nil {
		return
	}
	n += int64(v)

	return
}

// Len returns length of frame.
func (r WriteableResumeFrame) Len() int {
	return core.FrameHeaderLen + _lenTokenLength + _lenFirstPos + _lenLastRecvPos + _lenVersion + len(r.token)
}
