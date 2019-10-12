package framing

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/rsocket/rsocket-go/internal/common"
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
	return fmt.Sprintf("FrameLease{%s,timeToLive=%s,numberOfRequests=%d,metadata=%s}",
		p.header, p.TimeToLive(), p.NumberOfRequests(), string(p.Metadata()))
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

func NewFrameLease(ttl time.Duration, n uint32, metadata []byte) *FrameLease {
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
	}
	return &FrameLease{NewBaseFrame(NewFrameHeader(0, FrameTypeLease, fg), bf)}
}
