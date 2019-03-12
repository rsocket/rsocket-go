package rsocket

import "fmt"

var defaultFrameMetadataPushHeader = createHeader(0, tMetadataPush, flagMetadata)

type frameMetadataPush struct {
	*baseFrame
}

func (p *frameMetadataPush) String() string {
	return fmt.Sprintf("frameMetadataPush{%s,metadata=%s}", p.header, p.Metadata())
}

func (p *frameMetadataPush) Metadata() []byte {
	return p.body.Bytes()
}

func (p *frameMetadataPush) Data() []byte {
	return nil
}

func createMetadataPush(metadata []byte) *frameMetadataPush {
	bf := borrowByteBuffer()
	_, _ = bf.Write(metadata)
	return &frameMetadataPush{
		&baseFrame{
			header: defaultFrameMetadataPushHeader,
			body:   bf,
		},
	}
}
