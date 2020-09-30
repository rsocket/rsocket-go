package fragmentation

import (
	"github.com/rsocket/rsocket-go/core"
)

// HandleSplitResult is callback for fragmentation result.
type HandleSplitResult = func(index int, result SplitResult)

// SplitResult defines fragmentation result struct.
type SplitResult struct {
	Flag     core.FrameFlag
	Metadata []byte
	Data     []byte
}

// Split split data and metadata in frame.
func Split(mtu int, data []byte, metadata []byte, onFrame HandleSplitResult) {
	SplitSkip(mtu, 0, data, metadata, onFrame)
}

// SplitSkip skip some bytes and split data and metadata in frame.
func SplitSkip(mtu int, skip int, data []byte, metadata []byte, onFrame HandleSplitResult) {
	mlen, dlen := len(metadata), len(data)
	var idx, cursor1, cursor2 int
	var follow bool
	for {
		left := mtu - core.FrameHeaderLen
		if idx == 0 && skip > 0 {
			left -= skip
		}
		hasMetadata := cursor1 < mlen
		if hasMetadata {
			left -= 3
		}
		begin1, begin2 := cursor1, cursor2
		for wrote := 0; wrote < left; wrote++ {
			if cursor1 < mlen {
				cursor1++
			} else if cursor2 < dlen {
				cursor2++
			}
		}
		curMetadata := metadata[begin1:cursor1]
		curData := data[begin2:cursor2]
		follow = cursor1+cursor2 < mlen+dlen
		var flag core.FrameFlag
		if follow {
			flag |= core.FlagFollow
		} else {
			flag &= ^core.FlagFollow
		}
		if hasMetadata {
			// metadata
			flag |= core.FlagMetadata
		} else {
			// non-metadata
			flag &= ^core.FlagMetadata
		}
		onFrame(idx, SplitResult{
			Flag:     flag,
			Metadata: curMetadata,
			Data:     curData,
		})
		if !follow {
			break
		}
		idx++
	}
}
