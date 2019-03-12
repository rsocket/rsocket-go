package rsocket

import (
	"encoding/binary"
	"fmt"
)

type frameKeepalive struct {
	*baseFrame
}

func (p *frameKeepalive) String() string {
	return fmt.Sprintf("frameKeepalive{%s,lastReceivedPosition=%d,data=%s}", p.header, p.LastReceivedPosition(), p.Data())
}

func (p *frameKeepalive) LastReceivedPosition() uint64 {
	return binary.BigEndian.Uint64(p.body.Bytes())
}

func (p *frameKeepalive) Data() []byte {
	return p.body.Bytes()[8:]
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
