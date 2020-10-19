package fragmentation

import (
	"container/list"
	"fmt"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/u24"
	"github.com/rsocket/rsocket-go/payload"
)

const (
	// MinFragment is minimum fragment size in bytes.
	MinFragment = core.FrameHeaderLen + 4
	// MaxFragment is minimum fragment size in bytes.
	MaxFragment = u24.MaxUint24 - 3
)

var errInvalidFragmentLen = fmt.Errorf("invalid fragment: [%d,%d]", MinFragment, MaxFragment)

// HeaderAndPayload is Payload which having a FrameHeader.
type HeaderAndPayload interface {
	payload.Payload
	// FrameHeader returns a header of frame.
	Header() core.FrameHeader
}

// Joiner is used to join frames to a payload.
type Joiner interface {
	common.Releasable
	HeaderAndPayload
	// First returns the first frame.
	First() core.BufferedFrame
	// Push append a new frame and returns true if joiner is end.
	Push(elem HeaderAndPayload) (end bool)
}

// NewJoiner returns a new joiner.
func NewJoiner(first HeaderAndPayload) Joiner {
	root := list.New()
	root.PushBack(first)
	return &implJoiner{
		root: root,
	}
}

// IsValidFragment validate fragment size.
func IsValidFragment(fragment int) (err error) {
	if fragment < MinFragment || fragment > MaxFragment {
		err = errInvalidFragmentLen
	}
	return
}
