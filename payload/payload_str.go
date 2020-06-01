package payload

import (
	"strings"
)

type strPayload struct {
	data     string
	metadata string
}

func (p *strPayload) String() string {
	bu := strings.Builder{}
	bu.WriteString("Payload{data=")
	bu.WriteString(p.data)
	bu.WriteString("metadata=")
	bu.WriteString(p.metadata)
	bu.WriteByte('}')
	return bu.String()
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
