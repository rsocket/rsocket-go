package socket

import (
	"time"

	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/core/framing"
	"github.com/rsocket/rsocket-go/rx"
)

// SetupInfo represents basic info of setup.
type SetupInfo struct {
	Lease             bool
	Version           core.Version
	KeepaliveInterval time.Duration
	KeepaliveLifetime time.Duration
	Token             []byte
	DataMimeType      []byte
	Data              []byte
	MetadataMimeType  []byte
	Metadata          []byte
}

func (p *SetupInfo) toFrame() core.WriteableFrame {
	return framing.NewWriteableSetupFrame(
		p.Version,
		p.KeepaliveInterval,
		p.KeepaliveLifetime,
		p.Token,
		p.MetadataMimeType,
		p.DataMimeType,
		p.Data,
		p.Metadata,
		p.Lease,
	)
}

func tryRecover(e interface{}) (err error) {
	if e == nil {
		return
	}
	switch v := e.(type) {
	case error:
		err = v
	case string:
		err = errors.New(v)
	default:
		err = errors.Errorf("error: %s", v)
	}
	return
}

func ToIntRequestN(n uint32) int {
	if n > rx.RequestMax {
		return rx.RequestMax
	}
	return int(n)
}

func ToUint32RequestN(n int) uint32 {
	if n < 0 {
		panic("invalid negative int")
	}
	if n > rx.RequestMax {
		return rx.RequestMax
	}
	return uint32(n)
}
