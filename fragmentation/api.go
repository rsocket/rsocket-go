package fragmentation

import (
	"container/list"
	"fmt"

	"github.com/rsocket/rsocket-go/common"
	"github.com/rsocket/rsocket-go/framing"
	"github.com/rsocket/rsocket-go/payload"
)

const (
	// MinFragment is minimum fragment size in bytes.
	MinFragment = framing.HeaderLen + 4
	// MaxFragment is minimum fragment size in bytes.
	MaxFragment = common.MaxUint24 - 3
)

var errInvalidFragmentLen = fmt.Errorf("invalid fragment: [%d,%d]", MinFragment, MaxFragment)

// HeaderAndPayload is Payload which having a FrameHeader.
type HeaderAndPayload interface {
	payload.Payload
	// Header returns a header of frame.
	Header() framing.FrameHeader
}

// Splitter is used to split payload data and metadata to frames.
type Splitter interface {
	// Split split data and metadata to frames.
	Split(placeholder int, data []byte, metadata []byte, onFrame func(idx int, fg framing.FrameFlag, body *common.ByteBuff)) error
	// ShouldSplit returns the answer with given payload size.
	ShouldSplit(size int) bool
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

// NewSplitter returns a new splitter.
func NewSplitter(fragment int) (splitter Splitter, err error) {
	if fragment < MinFragment || fragment > MaxFragment {
		err = errInvalidFragmentLen
		return
	}
	splitter = implSplitter(fragment)
	return
}
