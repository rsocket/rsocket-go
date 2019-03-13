package rsocket

import (
	"encoding/binary"
)

const (
	lastRecvPosLen       = 8
	minKeepaliveFrameLen = lastRecvPosLen
)

type frameKeepalive struct {
	*baseFrame
}

func (p *frameKeepalive) validate() (err error) {
	if p.body.Len() < minKeepaliveFrameLen {
		err = errIncompleteFrame
	}

	return
}

func (p *frameKeepalive) LastReceivedPosition() uint64 {
	return binary.BigEndian.Uint64(p.body.Bytes())
}

func (p *frameKeepalive) Data() []byte {
	return p.body.Bytes()[lastRecvPosLen:]
}

func createKeepalive(position uint64, data []byte, respond bool) *frameKeepalive {
	var fg rFlags
	if respond {
		fg |= flagRespond
	}
	bf := borrowByteBuffer()
	for i := 0; i < 8; i++ {
		_ = bf.WriteByte(0)
	}
	binary.BigEndian.PutUint64(bf.Bytes(), position)
	if len(data) > 0 {
		_, _ = bf.Write(data)
	}
	return &frameKeepalive{
		&baseFrame{
			header: createHeader(0, tKeepalive, fg),
			body:   bf,
		},
	}
}
