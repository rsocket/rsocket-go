package payload

import (
	"fmt"

	"github.com/rsocket/rsocket-go/internal/common"
)

type rawPayload struct {
	data     []byte
	metadata []byte
}

func (p *rawPayload) String() string {
	m, _ := p.MetadataUTF8()
	return fmt.Sprintf("Payload{data=%s,metadata=%s}", p.DataUTF8(), m)
}

func (p *rawPayload) Metadata() (metadata []byte, ok bool) {
	return p.metadata, len(p.metadata) > 0
}

func (p *rawPayload) MetadataUTF8() (metadata string, ok bool) {
	raw, ok := p.Metadata()
	if ok {
		metadata = common.Bytes2str(raw)
	}
	return
}

func (p *rawPayload) Data() []byte {
	return p.data
}

func (p *rawPayload) DataUTF8() string {
	return common.Bytes2str(p.data)
}

func (*rawPayload) Release() {
	// ignore
}
