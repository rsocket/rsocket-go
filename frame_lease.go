package rsocket

import (
	"encoding/binary"
	"time"
)

type frameLease struct {
	*baseFrame
}

func (p *frameLease) TimeToLive() time.Duration {
	v := binary.BigEndian.Uint32(p.body.Bytes())
	return time.Millisecond * time.Duration(v)
}

func (p *frameLease) NumberOfRequests() uint32 {
	return binary.BigEndian.Uint32(p.body.Bytes()[4:])
}

func (p *frameLease) Metadata() []byte {
	if !p.header.Flag().Check(FlagMetadata) {
		return nil
	}
	return p.body.Bytes()[8:]
}
