package framing

import (
	"encoding/binary"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/bytebuffer"
	"github.com/rsocket/rsocket-go/internal/bytesconv"
	"github.com/rsocket/rsocket-go/internal/u24"
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
	size := core.FrameHeaderLen + 4 + len(data)

	if l := len(metadata); l > 0 {
		flag |= core.FlagMetadata
		size += 3 + l
	}
	bb := bytebuffer.BorrowByteBuff(size)

	if err := core.WriteFrameHeader(bb, sid, core.FrameTypeRequestChannel, flag); err != nil {
		bytebuffer.ReturnByteBuff(bb)
		panic(err)
	}

	if err := binary.Write(bb, binary.BigEndian, n); err != nil {
		bytebuffer.ReturnByteBuff(bb)
		panic(err)
	}

	if len(metadata) > 0 {
		if err := u24.WriteUint24(bb, len(metadata)); err != nil {
			bytebuffer.ReturnByteBuff(bb)
			panic(err)
		}
		if _, err := bb.Write(metadata); err != nil {
			bytebuffer.ReturnByteBuff(bb)
			panic(err)
		}
	}
	if len(data) > 0 {
		if _, err := bb.Write(data); err != nil {
			bytebuffer.ReturnByteBuff(bb)
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
	if r.HasFlag(core.FlagMetadata) && l < minRequestChannelFrameLen+3 {
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
	b, ok := r.Metadata()
	if ok {
		metadata = bytesconv.BytesToString(b)
	}
	return
}

// DataUTF8 returns data as UTF8 string.
func (r *RequestChannelFrame) DataUTF8() (data string) {
	b := r.Data()
	if len(b) > 0 {
		data = bytesconv.BytesToString(b)
	}
	return
}
