package framing

import (
	"encoding/binary"
	"io"
	"time"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

const (
	ttlLen        = 4
	reqOff        = ttlLen
	reqLen        = 4
	minLeaseFrame = ttlLen + reqLen
)

// LeaseFrame is Lease frame.
type LeaseFrame struct {
	*RawFrame
}

// WriteableLeaseFrame is writeable Lease frame.
type WriteableLeaseFrame struct {
	*tinyFrame
	ttl      [4]byte
	n        [4]byte
	metadata []byte
}

// Validate returns error if frame is invalid.
func (l *LeaseFrame) Validate() (err error) {
	if l.body.Len() < minLeaseFrame {
		err = errIncompleteFrame
	}
	return
}

// TimeToLive returns time to live duration.
func (l *LeaseFrame) TimeToLive() time.Duration {
	v := binary.BigEndian.Uint32(l.body.Bytes())
	return time.Millisecond * time.Duration(v)
}

// NumberOfRequests returns number of requests.
func (l *LeaseFrame) NumberOfRequests() uint32 {
	return binary.BigEndian.Uint32(l.body.Bytes()[reqOff:])
}

// Metadata returns metadata bytes.
func (l *LeaseFrame) Metadata() []byte {
	if !l.header.Flag().Check(core.FlagMetadata) {
		return nil
	}
	return l.body.Bytes()[8:]
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

// NewWriteableLeaseFrame creates a new WriteableLeaseFrame.
func NewWriteableLeaseFrame(ttl time.Duration, n uint32, metadata []byte) *WriteableLeaseFrame {
	var a, b [4]byte
	binary.BigEndian.PutUint32(a[:], uint32(int64(ttl)/1e6))
	binary.BigEndian.PutUint32(b[:], n)

	var flag core.FrameFlag
	if len(metadata) > 0 {
		flag |= core.FlagMetadata
	}
	h := core.NewFrameHeader(0, core.FrameTypeLease, flag)
	t := newTinyFrame(h)
	return &WriteableLeaseFrame{
		tinyFrame: t,
		ttl:       a,
		n:         b,
		metadata:  metadata,
	}
}

// NewLeaseFrame creates a new LeaseFrame.
func NewLeaseFrame(ttl time.Duration, n uint32, metadata []byte) *LeaseFrame {
	bf := common.NewByteBuff()
	if err := binary.Write(bf, binary.BigEndian, uint32(int64(ttl)/1e6)); err != nil {
		panic(err)
	}
	if err := binary.Write(bf, binary.BigEndian, n); err != nil {
		panic(err)
	}
	var fg core.FrameFlag
	if len(metadata) > 0 {
		fg |= core.FlagMetadata
		if _, err := bf.Write(metadata); err != nil {
			panic(err)
		}
	}
	return &LeaseFrame{NewRawFrame(core.NewFrameHeader(0, core.FrameTypeLease, fg), bf)}
}
