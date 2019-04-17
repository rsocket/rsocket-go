package fragmentation

import (
	"github.com/rsocket/rsocket-go/common"
	"github.com/rsocket/rsocket-go/framing"
)

type implSplitter int

func (s implSplitter) Split(placeholder int, data []byte, metadata []byte, onFrame func(idx int, fg framing.FrameFlag, body *common.ByteBuff)) (err error) {
	fragment := int(s)
	var bf *common.ByteBuff
	lenM, lenD := len(metadata), len(data)
	var cur1, cur2 int
	var idx int
	for {
		bf = common.BorrowByteBuffer()
		var wroteM int
		left := fragment - framing.HeaderLen
		if idx == 0 && placeholder > 0 {
			left -= placeholder
			for i := 0; i < placeholder; i++ {
				_ = bf.WriteByte(0)
			}
		}
		hasMetadata := cur1 < lenM
		if hasMetadata {
			left -= 3
			// write metadata length placeholder
			_ = bf.WriteUint24(0)
		}
		for wrote := 0; wrote < left; wrote++ {
			if cur1 < lenM {
				_ = bf.WriteByte(metadata[cur1])
				wroteM++
				cur1++
			} else if cur2 < lenD {
				_ = bf.WriteByte(data[cur2])
				cur2++
			}
		}
		follow := cur1+cur2 < lenM+lenD
		var fg framing.FrameFlag
		if follow {
			fg |= framing.FlagFollow
		} else {
			fg &= ^framing.FlagFollow
		}
		if wroteM > 0 {
			// set metadata length
			x := common.NewUint24(wroteM)
			for i := 0; i < len(x); i++ {
				if idx == 0 {
					bf.B[i+placeholder] = x[i]
				} else {
					bf.B[i] = x[i]
				}
			}
			fg |= framing.FlagMetadata
		} else {
			// non-metadata
			fg &= ^framing.FlagMetadata
		}
		if onFrame != nil {
			onFrame(idx, fg, bf)
		}
		if !follow {
			break
		}
		idx++
	}
	return
}

func (s implSplitter) ShouldSplit(size int) bool {
	return size > int(s)
}
