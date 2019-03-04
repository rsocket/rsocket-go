package rsocket

import (
	"encoding/binary"
	"fmt"
)

type frameRequestStream struct {
	*baseFrame
}

func (p *frameRequestStream) String() string {
	return fmt.Sprintf("frameRequestStream{%s,data=%s,metadata=%s,initialRequestN=%d}", p.header, p.Data(), p.Metadata(), p.InitialRequestN())
}

func (p *frameRequestStream) InitialRequestN() uint32 {
	return binary.BigEndian.Uint32(p.body.Bytes())
}

func (p *frameRequestStream) Metadata() []byte {
	return p.trySliceMetadata(4)
}

func (p *frameRequestStream) Data() []byte {
	return p.trySliceData(4)
}

func createRequestStream(id uint32, n uint32, data, metadata []byte, flags ...Flags) *frameRequestStream {
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
	return &frameRequestStream{
		&baseFrame{
			header: createHeader(id, tRequestStream, fg),
			body:   bf,
		},
	}
}
