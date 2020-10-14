package framing

import (
	"encoding/binary"
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
	*bufferedFrame
}

// NewLeaseFrame creates a new LeaseFrame.
func NewLeaseFrame(ttl time.Duration, n uint32, metadata []byte) *LeaseFrame {
	var fg core.FrameFlag
	if len(metadata) > 0 {
		fg |= core.FlagMetadata
	}

	b := common.BorrowByteBuff()

	if err := core.WriteFrameHeader(b, 0, core.FrameTypeLease, fg); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}

	ttlInMills := uint32(common.ToMilliseconds(ttl))
	if err := binary.Write(b, binary.BigEndian, ttlInMills); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}
	if err := binary.Write(b, binary.BigEndian, n); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}

	if len(metadata) > 0 {
		if _, err := b.Write(metadata); err != nil {
			common.ReturnByteBuff(b)
			panic(err)
		}
	}

	return &LeaseFrame{
		bufferedFrame: newBufferedFrame(b),
	}
}

// Validate returns error if frame is invalid.
func (l *LeaseFrame) Validate() (err error) {
	if l.bodyLen() < minLeaseFrame {
		err = errIncompleteFrame
	}
	return
}

// TimeToLive returns time to live duration.
func (l *LeaseFrame) TimeToLive() time.Duration {
	v := binary.BigEndian.Uint32(l.Body())
	return time.Millisecond * time.Duration(v)
}

// NumberOfRequests returns number of requests.
func (l *LeaseFrame) NumberOfRequests() uint32 {
	return binary.BigEndian.Uint32(l.Body()[reqOff:])
}

// Metadata returns metadata bytes.
func (l *LeaseFrame) Metadata() []byte {
	if !l.HasFlag(core.FlagMetadata) {
		return nil
	}
	return l.Body()[8:]
}
