package payload

import (
	"io/ioutil"
	"time"

	"github.com/rsocket/rsocket-go/internal/common"
)

type (
	// Payload is a stream message (upstream or downstream).
	// It contains data associated with a stream created by a previous request.
	// In Reactive Streams and Rx this is the 'onNext' event.
	Payload interface {
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
	}

	// SetupPayload is particular payload for RSocket Setup.
	SetupPayload interface {
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
)

// Clone create a copy of original payload.
func Clone(payload Payload) Payload {
	if payload == nil {
		return nil
	}
	switch v := payload.(type) {
	case *rawPayload:
		var data []byte
		if v.data != nil {
			data = make([]byte, len(v.data))
			copy(data, v.data)
		}
		var metadata []byte
		if v.metadata != nil {
			metadata = make([]byte, len(v.metadata))
			copy(metadata, v.metadata)
		}
		return &rawPayload{data: data, metadata: metadata}
	case *strPayload:
		return &strPayload{data: v.data, metadata: v.metadata}
	default:
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

}

// New create a new payload with bytes.
func New(data []byte, metadata []byte) Payload {
	return &rawPayload{
		data:     data,
		metadata: metadata,
	}
}

// NewString create a new payload with strings.
func NewString(data, metadata string) Payload {
	return &strPayload{
		data:     data,
		metadata: metadata,
	}
}

// NewFile create a new payload from file.
func NewFile(filename string, metadata []byte) (Payload, error) {
	bs, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return New(bs, metadata), nil
}

// MustNewFile create a new payload from file.
func MustNewFile(filename string, metadata []byte) Payload {
	foo, err := NewFile(filename, metadata)
	if err != nil {
		panic(err)
	}
	return foo
}
