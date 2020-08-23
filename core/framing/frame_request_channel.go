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
	*RawFrame
}

// WriteableRequestChannelFrame is writeable RequestChannel frame.
type WriteableRequestChannelFrame struct {
	*tinyFrame
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
	t := newTinyFrame(h)
	return &WriteableRequestChannelFrame{
		tinyFrame: t,
		n:         b,
		metadata:  metadata,
		data:      data,
	}
}

// NewRequestChannelFrame creates a new RequestChannelFrame.
func NewRequestChannelFrame(sid uint32, n uint32, data, metadata []byte, flag core.FrameFlag) *RequestChannelFrame {
	bf := common.NewByteBuff()
	var b4 [4]byte
	binary.BigEndian.PutUint32(b4[:], n)
	if _, err := bf.Write(b4[:]); err != nil {
		panic(err)
	}
	if len(metadata) > 0 {
		flag |= core.FlagMetadata
		if err := bf.WriteUint24(len(metadata)); err != nil {
			panic(err)
		}
		if _, err := bf.Write(metadata); err != nil {
			panic(err)
		}
	}
	if len(data) > 0 {
		if _, err := bf.Write(data); err != nil {
			panic(err)
		}
	}
	return &RequestChannelFrame{
		NewRawFrame(core.NewFrameHeader(sid, core.FrameTypeRequestChannel, flag), bf),
	}
}
