package rsocket

import (
	"encoding/binary"
	"fmt"
)

type frameRequestN struct {
	*baseFrame
}

func (p *frameRequestN) String() string {
	return fmt.Sprintf("frameRequestN{header=%s,n=%d}", p.header, p.N())
}

func (p *frameRequestN) N() uint32 {
	return binary.BigEndian.Uint32(p.body.Bytes())
}

func createRequestN(sid, n uint32, flags ...Flags) *frameRequestN {
	fg := newFlags(flags...)
	bf := borrowByteBuffer()
	for i := 0; i < 4; i++ {
		_ = bf.WriteByte(0)
	}
	binary.BigEndian.PutUint32(bf.Bytes(), n)
	return &frameRequestN{
		&baseFrame{
			header: createHeader(sid, tRequestN, fg),
			body:   bf,
		},
	}
}
