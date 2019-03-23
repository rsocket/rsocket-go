package payload

import (
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

// New create a new payload with bytes.
func New(data []byte, metadata []byte) Payload {
	return framing.NewFramePayload(0, data, metadata)
}

// NewString create a new payload with strings.
func NewString(data string, metadata string) Payload {
	return New([]byte(data), []byte(metadata))
}
