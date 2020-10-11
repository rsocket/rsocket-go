package framing

import (
	"encoding/binary"
	"io"
	"time"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

// WriteableLeaseFrame is writeable Lease frame.
type WriteableLeaseFrame struct {
	writeableFrame
	ttl      [4]byte
	n        [4]byte
	metadata []byte
}

// NewWriteableLeaseFrame creates a new WriteableLeaseFrame.
func NewWriteableLeaseFrame(ttl time.Duration, n uint32, metadata []byte) *WriteableLeaseFrame {
	var a, b [4]byte
	binary.BigEndian.PutUint32(a[:], uint32(common.ToMilliseconds(ttl)))
	binary.BigEndian.PutUint32(b[:], n)

	var flag core.FrameFlag
	if len(metadata) > 0 {
		flag |= core.FlagMetadata
	}
	h := core.NewFrameHeader(0, core.FrameTypeLease, flag)
	t := newWriteableFrame(h)
	return &WriteableLeaseFrame{
		writeableFrame: t,
		ttl:            a,
		n:              b,
		metadata:       metadata,
	}
}

// WriteTo writes frame to writer.
func (l WriteableLeaseFrame) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int64
	wrote, err = l.header.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote

	var v int
	v, err = w.Write(l.ttl[:])
	if err != nil {
		return
	}
	n += int64(v)

	v, err = w.Write(l.n[:])
	if err != nil {
		return
	}
	n += int64(v)

	if l.header.Flag().Check(core.FlagMetadata) {
		v, err = w.Write(l.metadata)
		if err != nil {
			return
		}
		n += int64(v)
	}

	return
}

// Len returns length of frame.
func (l WriteableLeaseFrame) Len() int {
	n := core.FrameHeaderLen + 8
	if l.header.Flag().Check(core.FlagMetadata) {
		n += len(l.metadata)
	}
	return n
}
