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
	var idx, cursor1, cursor2 int
	var follow bool
	for {
		bf = common.BorrowByteBuffer()
		var wroteM int
		left := fragment - framing.HeaderLen
		if idx == 0 && placeholder > 0 {
			left -= placeholder
			for i := 0; i < placeholder; i++ {
				if err := bf.WriteByte(0); err != nil {
					common.ReturnByteBuffer(bf)
					return err
				}
			}
		}
		hasMetadata := cursor1 < lenM
		if hasMetadata {
			left -= 3
			// write metadata length placeholder
			if err := bf.WriteUint24(0); err != nil {
				common.ReturnByteBuffer(bf)
				return err
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
		if _, err = bf.Write(metadata[begin1:cursor1]); err != nil {
			common.ReturnByteBuffer(bf)
			return err
		}
		if _, err := bf.Write(data[begin2:cursor2]); err != nil {
			common.ReturnByteBuffer(bf)
			return err
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
