package fragmentation

import (
	"container/list"
	"fmt"

	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/framing"
	"github.com/rsocket/rsocket-go/payload"
)

const (
	// MinFragment is minimum fragment size in bytes.
	MinFragment = framing.HeaderLen + 4
	// MaxFragment is minimum fragment size in bytes.
	MaxFragment = common.MaxUint24 - 3
)

var errInvalidFragmentLen = fmt.Errorf("invalid fragment: [%d,%d]", MinFragment, MaxFragment)

// HeaderAndPayload is Payload which having a Header.
type HeaderAndPayload interface {
	payload.Payload
	// Header returns a header of frame.
	Header() framing.Header
}

// Joiner is used to join frames to a payload.
type Joiner interface {
	HeaderAndPayload
	// First returns the first frame.
	First() framing.Frame
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
