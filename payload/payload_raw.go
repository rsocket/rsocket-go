package payload

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type rawPayload struct {
	data     []byte
	metadata []byte
}

func (p *rawPayload) String() string {
	bu := strings.Builder{}
	bu.WriteString("Payload{data=")
	if utf8.Valid(p.data) {
		bu.Write(p.data)
	} else {
		bu.WriteByte('[')
		for _, b := range p.data {
			bu.WriteString(fmt.Sprintf(" 0x%x", b))
		}
		bu.WriteByte(' ')
		bu.WriteByte(']')
	}
	bu.WriteString(",metadata=")
	if len(p.metadata) > 0 {
		if utf8.Valid(p.metadata) {
			bu.Write(p.metadata)
		} else {
			bu.WriteByte('[')
			for _, b := range p.metadata {
				bu.WriteString(fmt.Sprintf(" 0x%x", b))
			}
			bu.WriteByte(' ')
			bu.WriteByte(']')
		}
	}
	bu.WriteByte('}')
	return bu.String()
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
