package socket

import (
	"time"

	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/framing"
	"github.com/rsocket/rsocket-go/rx"
)

// SetupInfo represents basic info of setup.
type SetupInfo struct {
	Lease             bool
	Version           common.Version
	KeepaliveInterval time.Duration
	KeepaliveLifetime time.Duration
	Token             []byte
	DataMimeType      []byte
	Data              []byte
	MetadataMimeType  []byte
	Metadata          []byte
}

func (p *SetupInfo) toFrame() *framing.FrameSetup {
	return framing.NewFrameSetup(
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

func toIntN(n uint32) int {
	if n > rx.RequestMax {
		return rx.RequestMax
	}
	return int(n)
}

func toU32N(n int) uint32 {
	if n > rx.RequestMax {
		return rx.RequestMax
	}
	return uint32(n)
}
