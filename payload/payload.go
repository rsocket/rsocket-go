package payload

import (
	"fmt"
	"github.com/rsocket/rsocket-go/common"
	"github.com/rsocket/rsocket-go/framing"
	"time"
)

// Payload is a stream message (upstream or downstream).
// It contains data associated with a stream created by a previous request.
// In Reactive Streams and Rx this is the 'onNext' event.
type Payload interface {
	// Metadata returns raw metadata bytes.
	// The ok result indicates whether metadata exists.
	Metadata() (metadata []byte, ok bool)
	// MetadataUTF8 returns metadata as UTF8 string.
	// The ok result indicates whether metadata exists.
	MetadataUTF8() (metadata string, ok bool)
	// Data returns raw data bytes.
	Data() []byte
	// DataUTF8 returns data as UTF8 string.
	DataUTF8() string
	// Release release all resources of payload.
	// Some payload implements is pooled, so you must release resoures after using it.
	Release()
}

// SetupPayload is particular payload for RSocket Setup.
type SetupPayload interface {
	Payload
	// DataMimeType returns MIME type of data.
	DataMimeType() string
	// MetadataMimeType returns MIME type of metadata.
	MetadataMimeType() string
	// TimeBetweenKeepalive returns interval duration of keepalive.
	TimeBetweenKeepalive() time.Duration
	// MaxLifetime returns max lifetime of RSocket connection.
	MaxLifetime() time.Duration
	// Version return RSocket protocol version.
	Version() common.Version
}

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
		metadata = string(raw)
	}
	return
}

func (p *rawPayload) Data() []byte {
	return p.data
}

func (p *rawPayload) DataUTF8() (data string) {
	if len(p.data) > 0 {
		data = string(p.data)
	}
	return
}

func (*rawPayload) Release() {
	// ignore
}

type strPayload struct {
	data     string
	metadata string
}

func (p *strPayload) String() string {
	return fmt.Sprintf("Payload{data=%s,metadata=%s}", p.data, p.metadata)
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

func (*strPayload) Release() {
	// ignore
}

// Clone create a copy of original payload.
func Clone(payload Payload) Payload {
	ret := &rawPayload{}
	if d := payload.Data(); len(d) > 0 {
		clone := make([]byte, len(d))
		copy(clone, d)
		ret.data = clone
	}
	if m, ok := payload.Metadata(); ok && len(m) > 0 {
		clone := make([]byte, len(m))
		copy(clone, m)
		ret.metadata = clone
	}
	return ret
}

// New create a new payload with bytes.
func New(data []byte, metadata []byte) Payload {
	return &rawPayload{
		data:     data,
		metadata: metadata,
	}
}

// NewPooled returns a pooled payload.
// Remember call Release() at last.
// Sometimes payload will be released automatically. (eg: sent by a requester).
func NewPooled(data, metadata []byte) Payload {
	return framing.NewFramePayload(0, data, metadata)
}

// NewString create a new payload with strings.
func NewString(data, metadata string) Payload {
	return &strPayload{
		data:     data,
		metadata: metadata,
	}
}
