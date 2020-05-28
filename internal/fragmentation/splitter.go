package fragmentation

import (
	"github.com/rsocket/rsocket-go/internal/common"
	"github.com/rsocket/rsocket-go/internal/framing"
)

type splitResult struct {
	f framing.FrameFlag
	b *common.ByteBuff
}

// Split split data and metadata in frame.
func Split(mtu int, data []byte, metadata []byte, onFrame func(idx int, fg framing.FrameFlag, body *common.ByteBuff)) {
	SplitSkip(mtu, 0, data, metadata, onFrame)
}

// SplitSkip skip some bytes and split data and metadata in frame.
func SplitSkip(mtu int, skip int, data []byte, metadata []byte, onFrame func(idx int, fg framing.FrameFlag, body *common.ByteBuff)) {
	ch := make(chan splitResult, 3)
	go func(mtu int, skip int, data []byte, metadata []byte, ch chan splitResult) {
		defer func() {
			close(ch)
		}()
		var bf *common.ByteBuff
		lenM, lenD := len(metadata), len(data)
		var idx, cursor1, cursor2 int
		var follow bool
		for {
			bf = common.NewByteBuff()
			var wroteM int
			left := mtu - framing.HeaderLen
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
				// write metadata length placeholder
				if err := bf.WriteUint24(0); err != nil {
					panic(err)
				}
			}
			begin1, begin2 := cursor1, cursor2
			for wrote := 0; wrote < left; wrote++ {
				if cursor1 < lenM {
					wroteM++
					cursor1++
				} else if cursor2 < lenD {
					cursor2++
				}
			}
			if _, err := bf.Write(metadata[begin1:cursor1]); err != nil {
				panic(err)
			}
			if _, err := bf.Write(data[begin2:cursor2]); err != nil {
				panic(err)
			}
			follow = cursor1+cursor2 < lenM+lenD
			var fg framing.FrameFlag
			if follow {
				fg |= framing.FlagFollow
			} else {
				fg &= ^framing.FlagFollow
			}
			if wroteM > 0 {
				// set metadata length
				x := common.MustNewUint24(wroteM)
				for i := 0; i < len(x); i++ {
					if idx == 0 {
						bf.Bytes()[i+skip] = x[i]
					} else {
						bf.Bytes()[i] = x[i]
					}
				}
				fg |= framing.FlagMetadata
			} else {
				// non-metadata
				fg &= ^framing.FlagMetadata
			}
			ch <- splitResult{
				f: fg,
				b: bf,
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
			onFrame(idx, v.f, v.b)
		}
		idx++
	}
}
