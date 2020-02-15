package extension

import (
	"fmt"

	"github.com/pkg/errors"
)

var errTagLengthExceed = errors.New("length of tag exceed 255")

// EncodeRouting encode routing tags to raw bytes.
func EncodeRouting(tag string, otherTags ...string) (raw []byte, err error) {
	// process first
	size := len(tag)
	if size > 0xFF {
		raw = nil
		err = errTagLengthExceed
		return
	}
	raw = append(raw, byte(len(tag)))
	raw = append(raw, tag...)
	// process others
	for _, it := range otherTags {
		size = len(it)
		if size > 0xFF {
			raw = nil
			err = errTagLengthExceed
			return
		}
		raw = append(raw, byte(len(it)))
		raw = append(raw, it...)
	}
	return
}

// ParseRoutingTags parse routing tags in metadata.
// See: https://github.com/rsocket/rsocket/blob/master/Extensions/Routing.md
func ParseRoutingTags(bs []byte) (tags []string, err error) {
	totals := len(bs)
	cursor := 0
	for {
		if cursor >= totals {
			break
		}
		tagLen := int(bs[cursor])
		end := cursor + 1 + tagLen
		if end > totals {
			err = fmt.Errorf("bad routing tags: illegal tag len %d", tagLen)
			return
		}
		tags = append(tags, string(bs[cursor+1:end]))
		cursor += tagLen + 1
	}
	return
}
