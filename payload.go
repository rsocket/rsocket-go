package rsocket

import "time"

// Payload is a stream message (upstream or downstream).
// It contains data associated with a stream created by a previous request.
// In Reactive Streams and Rx this is the 'onNext' event.
type Payload interface {
	// Metadata returns raw metadata bytes.
	Metadata() []byte
	// Data returns raw data bytes.
	Data() []byte
	// Release release all resources of payload.
	// Some payload implements is pooled, so you must release resoures after using it.
	Release()
}

// NewPayload create a new payload.
func NewPayload(data []byte, metadata []byte) Payload {
	return createPayloadFrame(0, data, metadata)
}

// NewPayloadString create a new payload with strings.
func NewPayloadString(data string, metadata string) Payload {
	return NewPayload([]byte(data), []byte(metadata))
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
	Version() Version
}
