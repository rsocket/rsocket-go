package fragmentation

import (
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
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
	ch := make(chan SplitResult, 3)
	go func(mtu int, skip int, data []byte, metadata []byte, ch chan SplitResult) {
		defer func() {
			close(ch)
		}()
		var bf *common.ByteBuff
		lenM, lenD := len(metadata), len(data)
		var idx, cursor1, cursor2 int
		var follow bool
		for {
			bf = common.NewByteBuff()
			left := mtu - core.FrameHeaderLen
			if idx == 0 && skip > 0 {
				left -= skip
				for i := 0; i < skip; i++ {
					if err := bf.WriteByte(0); err != nil {
						panic(err)
					}
				}
			}
			hasMetadata := cursor1 < lenM
			if hasMetadata {
				left -= 3
			}
			begin1, begin2 := cursor1, cursor2
			for wrote := 0; wrote < left; wrote++ {
				if cursor1 < lenM {
					cursor1++
				} else if cursor2 < lenD {
					cursor2++
				}
			}
			curMetadata := metadata[begin1:cursor1]
			curData := data[begin2:cursor2]
			follow = cursor1+cursor2 < lenM+lenD
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
			ch <- SplitResult{
				Flag:     flag,
				Metadata: curMetadata,
				Data:     curData,
			}
			if !follow {
				break
			}
			idx++
		}
	}(mtu, skip, data, metadata, ch)
	var idx int
	for v := range ch {
		if onFrame != nil {
			onFrame(idx, v)
		}
		idx++
	}
}
