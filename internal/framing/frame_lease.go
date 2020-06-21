package framing

import (
	"encoding/binary"
	"io"
	"time"

	"github.com/rsocket/rsocket-go/internal/common"
)

const (
	ttlLen        = 4
	reqOff        = ttlLen
	reqLen        = 4
	minLeaseFrame = ttlLen + reqLen
)

// LeaseFrame is lease frame.
type LeaseFrame struct {
	*RawFrame
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
	if !l.header.Flag().Check(FlagMetadata) {
		return nil
	}
	return l.body.Bytes()[8:]
}

type LeaseFrameSupport struct {
	*tinyFrame
	ttl      [4]byte
	n        [4]byte
	metadata []byte
}

func (l LeaseFrameSupport) WriteTo(w io.Writer) (n int64, err error) {
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

	if l.header.Flag().Check(FlagMetadata) {
		v, err = w.Write(l.metadata)
		if err != nil {
			return
		}
		n += int64(v)
	}

	return
}

func (l LeaseFrameSupport) Len() int {
	n := HeaderLen + 8
	if l.header.Flag().Check(FlagMetadata) {
		n += len(l.metadata)
	}
	return n
}

func NewLeaseFrameSupport(ttl time.Duration, n uint32, metadata []byte) *LeaseFrameSupport {
	var a, b [4]byte
	binary.BigEndian.PutUint32(a[:], uint32(ttl.Milliseconds()))
	binary.BigEndian.PutUint32(b[:], n)

	var flag FrameFlag
	if len(metadata) > 0 {
		flag |= FlagMetadata
	}
	h := NewFrameHeader(0, FrameTypeLease, flag)
	t := newTinyFrame(h)
	return &LeaseFrameSupport{
		tinyFrame: t,
		ttl:       a,
		n:         b,
		metadata:  metadata,
	}
}

func NewLeaseFrame(ttl time.Duration, n uint32, metadata []byte) *LeaseFrame {
	bf := common.NewByteBuff()
	if err := binary.Write(bf, binary.BigEndian, uint32(ttl.Milliseconds())); err != nil {
		panic(err)
	}
	if err := binary.Write(bf, binary.BigEndian, n); err != nil {
		panic(err)
	}
	var fg FrameFlag
	if len(metadata) > 0 {
		fg |= FlagMetadata
		if _, err := bf.Write(metadata); err != nil {
			panic(err)
		}
	}
	return &LeaseFrame{NewRawFrame(NewFrameHeader(0, FrameTypeLease, fg), bf)}
}
