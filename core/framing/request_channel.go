package framing

import (
	"encoding/binary"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

const (
	initReqLen                = 4
	minRequestChannelFrameLen = initReqLen
)

// RequestChannelFrame is RequestChannel frame.
type RequestChannelFrame struct {
	*bufferedFrame
}

// NewRequestChannelFrame creates a new RequestChannelFrame.
func NewRequestChannelFrame(sid uint32, n uint32, data, metadata []byte, flag core.FrameFlag) *RequestChannelFrame {
	if len(metadata) > 0 {
		flag |= core.FlagMetadata
	}
	bb := common.BorrowByteBuff()

	if err := core.WriteFrameHeader(bb, sid, core.FrameTypeRequestChannel, flag); err != nil {
		common.ReturnByteBuff(bb)
		panic(err)
	}

	if err := binary.Write(bb, binary.BigEndian, n); err != nil {
		common.ReturnByteBuff(bb)
		panic(err)
	}

	if len(metadata) > 0 {
		if err := bb.WriteUint24(len(metadata)); err != nil {
			common.ReturnByteBuff(bb)
			panic(err)
		}
		if _, err := bb.Write(metadata); err != nil {
			common.ReturnByteBuff(bb)
			panic(err)
		}
	}
	if len(data) > 0 {
		if _, err := bb.Write(data); err != nil {
			common.ReturnByteBuff(bb)
			panic(err)
		}
	}

	return &RequestChannelFrame{
		bufferedFrame: newBufferedFrame(bb),
	}
}

// Validate returns error if frame is invalid.
func (r *RequestChannelFrame) Validate() error {
	l := r.bodyLen()
	if l < minRequestChannelFrameLen {
		return errIncompleteFrame
	}
	if r.Header().Flag().Check(core.FlagMetadata) && l < minRequestChannelFrameLen+3 {
		return errIncompleteFrame
	}
	return nil
}

// InitialRequestN returns initial N.
func (r *RequestChannelFrame) InitialRequestN() uint32 {
	return binary.BigEndian.Uint32(r.Body())
}

// Metadata returns metadata bytes.
func (r *RequestChannelFrame) Metadata() ([]byte, bool) {
	return r.trySliceMetadata(initReqLen)
}

// Data returns data bytes.
func (r *RequestChannelFrame) Data() []byte {
	return r.trySliceData(initReqLen)
}

// MetadataUTF8 returns metadata as UTF8 string.
func (r *RequestChannelFrame) MetadataUTF8() (metadata string, ok bool) {
	raw, ok := r.Metadata()
	if ok {
		metadata = string(raw)
	}
	return
}

// DataUTF8 returns data as UTF8 string.
func (r *RequestChannelFrame) DataUTF8() string {
	return string(r.Data())
}
