package payload

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/framing"
)

var (
	errNotFile               = errors.New("target is not file")
	errTooLargePooledPayload = fmt.Sprintf("too large pooled payload: maximum size is %d", common.MaxUint24-3)
)

type (
	Releasable interface {
		// Release release all resources of payload.
		// Some payload implements is pooled, so you must release resoures after using it.
		Release()
	}
	// Payload is a stream message (upstream or downstream).
	// It contains data associated with a stream created by a previous request.
	// In Reactive Streams and Rx this is the 'onNext' event.
	Payload interface {
		Releasable
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
	size := framing.CalcPayloadFrameSize(data, metadata)
	if size > common.MaxUint24-3 {
		panic(errTooLargePooledPayload)
	}
	return framing.NewFramePayload(0, data, metadata)
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
	fi, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}
	if fi.IsDir() {
		return nil, errNotFile
	}
	// Check file size
	//if fi.Size() > common.MaxUint24 {
	//	return nil, errTooLargeFile
	//}
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
