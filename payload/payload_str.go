package payload

import (
	"github.com/rsocket/rsocket-go/internal/bytesconv"
)

type strPayload struct {
	data     string
	metadata string
}

func (p *strPayload) Metadata() (metadata []byte, ok bool) {
	ok = len(p.metadata) > 0
	if ok {
		metadata = bytesconv.StringToBytes(p.metadata)
	}
	return
}

func (p *strPayload) MetadataUTF8() (metadata string, ok bool) {
	return p.metadata, len(p.metadata) > 0
}

func (p *strPayload) Data() []byte {
	return bytesconv.StringToBytes(p.data)
}

func (p *strPayload) DataUTF8() string {
	return p.data
}
