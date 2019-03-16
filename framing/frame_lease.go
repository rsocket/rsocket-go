package framing

import (
	"encoding/binary"
	"fmt"
	"time"
)

const (
	ttlLen        = 4
	reqOff        = ttlLen
	reqLen        = 4
	minLeaseFrame = ttlLen + reqLen
)

// FrameLease is lease frame.
type FrameLease struct {
	*BaseFrame
}

// Validate returns error if frame is invalid.
func (p *FrameLease) Validate() (err error) {
	if p.body.Len() < minLeaseFrame {
		err = errIncompleteFrame
	}
	return
}

func (p *FrameLease) String() string {
	return fmt.Sprintf("FrameLease{%s,timeToLive=%d,numberOfRequests=%d,metadata=%s}", p.header, p.TimeToLive(), p.NumberOfRequests(), p.Metadata())
}

// TimeToLive returns time to live duration.
func (p *FrameLease) TimeToLive() time.Duration {
	v := binary.BigEndian.Uint32(p.body.Bytes())
	return time.Millisecond * time.Duration(v)
}

// NumberOfRequests returns number of requests.
func (p *FrameLease) NumberOfRequests() uint32 {
	return binary.BigEndian.Uint32(p.body.Bytes()[reqOff:])
}

// Metadata returns metadata bytes.
func (p *FrameLease) Metadata() []byte {
	if !p.header.Flag().Check(FlagMetadata) {
		return nil
	}
	return p.body.Bytes()[8:]
}
