package rsocket

var defaultFrameMetadataPushHeader = createHeader(0, tMetadataPush, FlagMetadata)

type frameMetadataPush struct {
	*baseFrame
}

func (p *frameMetadataPush) Metadata() []byte {
	return p.body.Bytes()
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
