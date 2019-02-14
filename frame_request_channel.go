package rsocket

import (
	"encoding/binary"
	"fmt"
)

type frameRequestChannel struct {
	*baseFrame
}

func (p *frameRequestChannel) String() string {
	return fmt.Sprintf("frameRequestChannel{%s,data=%s,metadata=%s,initialRequestN=%d}", p.header, p.Data(), p.Metadata(), p.InitialRequestN())
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

func createRequestChannel(sid uint32, n uint32, data, metadata []byte, flags ...Flags) *frameRequestChannel {
	fg := newFlags(flags...)
	bf := borrowByteBuffer()
	for range [4]struct{}{} {
		_ = bf.WriteByte(0)
	}
	binary.BigEndian.PutUint32(bf.Bytes(), n)
	if len(metadata) > 0 {
		fg |= FlagMetadata
		_ = bf.WriteUint24(len(metadata))
		_, _ = bf.Write(metadata)
	}
	if len(data) > 0 {
		_, _ = bf.Write(data)
	}
	return &frameRequestChannel{
		&baseFrame{
			header: createHeader(sid, tRequestChannel, fg),
			body:   bf,
		},
	}
}
