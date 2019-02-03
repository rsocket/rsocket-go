package rsocket

import (
	"encoding/binary"
	"fmt"
)

type frameRequestChannel struct {
	*baseFrame
}

func (p *frameRequestChannel) String() string {
	return fmt.Sprintf("frameRequestChannel{%s,data=%s,metadata=%s,initialRequestN=%d}", p.header, string(p.Data()), string(p.Metadata()), p.InitialRequestN())
}

func (p *frameRequestChannel) InitialRequestN() uint32 {
	return binary.BigEndian.Uint32(p.body.Bytes())
}

func (p *frameRequestChannel) Metadata() []byte {
	m, _ := extractMetadataAndData(p.header, p.body.Bytes()[4:])
	return m
}

func (p *frameRequestChannel) Data() []byte {
	_, d := extractMetadataAndData(p.header, p.body.Bytes()[4:])
	return d
}
