package socket

import (
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/framing"
	"github.com/rsocket/rsocket-go/rx"
)

// SetupInfo represents basic info of setup.
type SetupInfo struct {
	Version           common.Version
	KeepaliveInterval time.Duration
	KeepaliveLifetime time.Duration
	Token             []byte
	DataMimeType      []byte
	Data              []byte
	MetadataMimeType  []byte
	Metadata          []byte
}

// ToFrame converts current SetupInfo to a frame of Setup.
func (p *SetupInfo) ToFrame() *framing.FrameSetup {
	return framing.NewFrameSetup(
		p.Version,
		p.KeepaliveInterval,
		p.KeepaliveLifetime,
		p.Token,
		p.MetadataMimeType,
		p.DataMimeType,
		p.Data,
		p.Metadata,
	)
}

type streamIDs interface {
	next() uint32
}

type serverStreamIDs struct {
	cur uint32
}

func (p *serverStreamIDs) next() uint32 {
	// 2,4,6,8...
	v := 2 * atomic.AddUint32(&p.cur, 1)
	if v != 0 {
		return v
	}
	return p.next()
}

type clientStreamIDs struct {
	cur uint32
}

func (p *clientStreamIDs) next() uint32 {
	// 1,3,5,7
	v := 2*(atomic.AddUint32(&p.cur, 1)-1) + 1
	if v != 0 {
		return v
	}
	return p.next()
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
