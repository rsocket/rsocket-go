package rsocket

import "encoding/binary"

type frameRequestStream struct {
	*baseFrame
}

func (p *frameRequestStream) InitialRequestN() uint32 {
	return binary.BigEndian.Uint32(p.body.Bytes())
}

func (p *frameRequestStream) Metadata() []byte {
	m, _ := extractMetadataAndData(p.header, p.body.Bytes()[4:])
	return m
}

func (p *frameRequestStream) Data() []byte {
	_, d := extractMetadataAndData(p.header, p.body.Bytes()[4:])
	return d
}
