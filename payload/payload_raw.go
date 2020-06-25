package payload

type rawPayload struct {
	data     []byte
	metadata []byte
}

func (p *rawPayload) Metadata() (metadata []byte, ok bool) {
	return p.metadata, len(p.metadata) > 0
}

func (p *rawPayload) MetadataUTF8() (metadata string, ok bool) {
	raw, ok := p.Metadata()
	if ok {
		metadata = string(raw)
	}
	return
}

func (p *rawPayload) Data() []byte {
	return p.data
}

func (p *rawPayload) DataUTF8() string {
	return string(p.data)
}
