package framing

import (
	"encoding/binary"
	"io"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

const (
	initReqLen                = 4
	minRequestChannelFrameLen = initReqLen
)

// RequestChannelFrame is RequestChannel frame.
type RequestChannelFrame struct {
	*baseDefaultFrame
}

// WriteableRequestChannelFrame is writeable RequestChannel frame.
type WriteableRequestChannelFrame struct {
	baseWriteableFrame
	n        [4]byte
	metadata []byte
	data     []byte
}

// Validate returns error if frame is invalid.
func (r *RequestChannelFrame) Validate() error {
	l := r.body.Len()
	if l < minRequestChannelFrameLen {
		return errIncompleteFrame
	}
	if r.header.Flag().Check(core.FlagMetadata) && l < minRequestChannelFrameLen+3 {
		return errIncompleteFrame
	}
	return nil
}

// InitialRequestN returns initial N.
func (r *RequestChannelFrame) InitialRequestN() uint32 {
	return binary.BigEndian.Uint32(r.body.Bytes())
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

// WriteTo writes frame to writer.
func (r WriteableRequestChannelFrame) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int64
	wrote, err = r.header.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote

	var v int
	v, err = w.Write(r.n[:])
	if err != nil {
		return
	}
	n += int64(v)

	wrote, err = writePayload(w, r.data, r.metadata)
	if err != nil {
		return
	}
	n += wrote

	return
}

// Len returns length of frame.
func (r WriteableRequestChannelFrame) Len() int {
	return CalcPayloadFrameSize(r.data, r.metadata) + 4
}

// NewWriteableRequestChannelFrame creates a new WriteableRequestChannelFrame.
func NewWriteableRequestChannelFrame(sid uint32, n uint32, data, metadata []byte, flag core.FrameFlag) *WriteableRequestChannelFrame {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], n)
	if len(metadata) > 0 {
		flag |= core.FlagMetadata
	}
	h := core.NewFrameHeader(sid, core.FrameTypeRequestChannel, flag)
	t := newBaseWriteableFrame(h)
	return &WriteableRequestChannelFrame{
		baseWriteableFrame: t,
		n:                  b,
		metadata:           metadata,
		data:               data,
	}
}

// NewRequestChannelFrame creates a new RequestChannelFrame.
func NewRequestChannelFrame(sid uint32, n uint32, data, metadata []byte, flag core.FrameFlag) *RequestChannelFrame {
	b := common.BorrowByteBuff()

	if err := binary.Write(b, binary.BigEndian, n); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}

	if len(metadata) > 0 {
		flag |= core.FlagMetadata
		if err := b.WriteUint24(len(metadata)); err != nil {
			common.ReturnByteBuff(b)
			panic(err)
		}
		if _, err := b.Write(metadata); err != nil {
			common.ReturnByteBuff(b)
			panic(err)
		}
	}
	if len(data) > 0 {
		if _, err := b.Write(data); err != nil {
			common.ReturnByteBuff(b)
			panic(err)
		}
	}
	return &RequestChannelFrame{
		newBaseDefaultFrame(core.NewFrameHeader(sid, core.FrameTypeRequestChannel, flag), b),
	}
}
