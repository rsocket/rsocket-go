package rsocket

type frameRequestResponse struct {
	*baseFrame
}

func (p *frameRequestResponse) Metadata() []byte {
	m, _ := extractMetadataAndData(p.header, p.body.Bytes())
	return m
}
func (p *frameRequestResponse) Data() []byte {
	_, d := extractMetadataAndData(p.header, p.body.Bytes())
	return d
}

func createRequestResponse(id uint32, data, metadata []byte, flags ...Flags) *frameRequestResponse {
	fg := newFlags(flags...)
	bf := borrowByteBuffer()
	if len(metadata) > 0 {
		fg |= FlagMetadata
		_ = bf.WriteUint24(len(metadata))
		_, _ = bf.Write(metadata)
	}
	if len(data) > 0 {
		_, _ = bf.Write(data)
	}
	return &frameRequestResponse{
		&baseFrame{
			header: createHeader(id, tRequestResponse, fg),
			body:   bf,
		},
	}
}
