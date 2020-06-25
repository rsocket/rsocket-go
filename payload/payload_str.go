package payload

type strPayload struct {
	data     string
	metadata string
}

func (p *strPayload) Metadata() (metadata []byte, ok bool) {
	ok = len(p.metadata) > 0
	if ok {
		metadata = []byte(p.metadata)
	}
	return
}

func (p *strPayload) MetadataUTF8() (metadata string, ok bool) {
	return p.metadata, len(p.metadata) > 0
}

func (p *strPayload) Data() []byte {
	return []byte(p.data)
}

func (p *strPayload) DataUTF8() string {
	return p.data
}
