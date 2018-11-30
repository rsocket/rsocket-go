package rsocket

import (
	"encoding/binary"
	"time"
)

type FrameLease struct {
	*Header
	timeToLive       time.Duration
	numberOfRequests uint32
	metadata         []byte
}

func (p *FrameLease) TimeToLive() time.Duration {
	return p.timeToLive
}

func (p *FrameLease) NumberOfRequests() uint32 {
	return p.numberOfRequests
}

func (p *FrameLease) Metadata() []byte {
	return p.metadata
}

func asLease(h *Header, raw []byte) *FrameLease {
	t1 := binary.BigEndian.Uint32(raw[frameHeaderLength : frameHeaderLength+4])
	n := binary.BigEndian.Uint32(raw[frameHeaderLength+4 : frameHeaderLength+8])
	return &FrameLease{
		Header:           h,
		timeToLive:       time.Millisecond * time.Duration(t1),
		numberOfRequests: n,
		metadata:         sliceMetadata(h, raw, frameHeaderLength+8),
	}
}

func mkLease(sid uint32, ttl time.Duration, requests uint32, meatadata []byte, f ...Flags) *FrameLease {
	return &FrameLease{
		Header:           mkHeader(sid, LEASE, f...),
		timeToLive:       ttl,
		numberOfRequests: requests,
		metadata:         meatadata,
	}
}
