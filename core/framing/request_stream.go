package framing

import (
	"encoding/binary"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

const (
	minRequestStreamFrameLen = initReqLen
)

// RequestStreamFrame is RequestStream frame.
type RequestStreamFrame struct {
	*bufferedFrame
}

// NewRequestStreamFrame returns a new RequestStreamFrame.
func NewRequestStreamFrame(id uint32, n uint32, data, metadata []byte, flag core.FrameFlag) *RequestStreamFrame {
	if len(metadata) > 0 {
		flag |= core.FlagMetadata
	}

	bb := common.BorrowByteBuff()

	if err := core.WriteFrameHeader(bb, id, core.FrameTypeRequestStream, flag); err != nil {
		common.ReturnByteBuff(bb)
		panic(err)
	}

	if err := binary.Write(bb, binary.BigEndian, n); err != nil {
		common.ReturnByteBuff(bb)
		panic(err)
	}
	if len(metadata) > 0 {
		flag |= core.FlagMetadata
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

	return &RequestStreamFrame{
		bufferedFrame: newBufferedFrame(bb),
	}
}

// Validate returns error if frame is invalid.
func (r *RequestStreamFrame) Validate() error {
	l := r.bodyLen()
	if l < minRequestStreamFrameLen {
		return errIncompleteFrame
	}
	if r.Header().Flag().Check(core.FlagMetadata) && l < minRequestStreamFrameLen+3 {
		return errIncompleteFrame
	}
	return nil
}

// InitialRequestN returns initial request N.
func (r *RequestStreamFrame) InitialRequestN() uint32 {
	return binary.BigEndian.Uint32(r.Body())
}

// Metadata returns metadata bytes.
func (r *RequestStreamFrame) Metadata() ([]byte, bool) {
	return r.trySliceMetadata(4)
}

// Data returns data bytes.
func (r *RequestStreamFrame) Data() []byte {
	return r.trySliceData(4)
}

// MetadataUTF8 returns metadata as UTF8 string.
func (r *RequestStreamFrame) MetadataUTF8() (metadata string, ok bool) {
	raw, ok := r.Metadata()
	if ok {
		metadata = string(raw)
	}
	return
}

// DataUTF8 returns data as UTF8 string.
func (r *RequestStreamFrame) DataUTF8() string {
	return string(r.Data())
}
