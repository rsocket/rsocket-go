package rsocket

import (
	"encoding/binary"
	"io"
	"time"
)

type FrameLease struct {
	*Header
	timeToLive       uint32
	numberOfRequests uint32
	metadata         []byte
}

func (p *FrameLease) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int
	wrote, err = w.Write(p.Header.Bytes())
	n += int64(wrote)
	if err != nil {
		return
	}
	b4 := make([]byte, 4)
	binary.BigEndian.PutUint32(b4, p.timeToLive)
	wrote, err = w.Write(b4)
	n += int64(wrote)
	if err != nil {
		return
	}
	binary.BigEndian.PutUint32(b4, p.numberOfRequests)
	wrote, err = w.Write(b4)
	n += int64(wrote)
	if err != nil {
		return
	}
	if !p.Header.Flags().Check(FlagMetadata) {
		return
	}
	wrote, err = w.Write(p.metadata)
	n += int64(wrote)
	return
}

func (p *FrameLease) Size() int {
	size := headerLen + 8
	if p.Header.Flags().Check(FlagMetadata) {
		size += len(p.metadata)
	}
	return size
}

func (p *FrameLease) TimeToLive() time.Duration {
	return time.Millisecond * time.Duration(p.timeToLive)
}

func (p *FrameLease) NumberOfRequests() uint32 {
	return p.numberOfRequests
}

func (p *FrameLease) Metadata() []byte {
	return p.metadata
}

func asLease(h *Header, raw []byte) *FrameLease {
	t1 := binary.BigEndian.Uint32(raw[headerLen : headerLen+4])
	n := binary.BigEndian.Uint32(raw[headerLen+4 : headerLen+8])
	var metadata []byte
	if h.Flags().Check(FlagMetadata) {
		foo := raw[headerLen+8:]
		metadata = make([]byte, len(foo))
		copy(metadata, foo)
	}
	return &FrameLease{
		Header:           h,
		timeToLive:       t1,
		numberOfRequests: n,
		metadata:         metadata,
	}
}

func mkLease(sid uint32, ttl time.Duration, requests uint32, meatadata []byte, f ...Flags) *FrameLease {
	return &FrameLease{
		Header:           mkHeader(sid, LEASE, f...),
		timeToLive:       uint32(ttl.Nanoseconds() / 1e6),
		numberOfRequests: requests,
		metadata:         meatadata,
	}
}
