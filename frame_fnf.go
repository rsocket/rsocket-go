package rsocket

import "fmt"

type frameFNF struct {
	*baseFrame
}

func (p *frameFNF) String() string {
	return fmt.Sprintf("frameFNF{%s,data=%s,metadata=%s}", p.header, p.Data(), p.Metadata())
}

func (p *frameFNF) Metadata() []byte {
	return p.trySliceMetadata(0)
}

func (p *frameFNF) Data() []byte {
	return p.trySliceData(0)
}

func createFNF(sid uint32, data, metadata []byte, flags ...Flags) *frameFNF {
	fg := newFlags(flags...)
	bf := borrowByteBuffer()
	if len(metadata) > 0 {
		fg |= FlagMetadata
		_ = bf.WriteUint24(len(metadata))
		_, _ = bf.Write(metadata)
	}
	_, _ = bf.Write(data)
	return &frameFNF{
		&baseFrame{
			header: createHeader(sid, tRequestFNF, fg),
			body:   bf,
		},
	}
}
