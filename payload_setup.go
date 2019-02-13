package rsocket

import "time"

type SetupPayload interface {
	Payload
	DataMimeType() string
	MetadataMimeType() string
	TimeBetweenKeepalive() time.Duration
	MaxLifetime() time.Duration
	Version() Version
}
