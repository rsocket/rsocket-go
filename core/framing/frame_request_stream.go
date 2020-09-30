package framing

import (
	"encoding/binary"
	"io"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

const (
	minRequestStreamFrameLen = initReqLen
)

// RequestStreamFrame is RequestStream frame.
type RequestStreamFrame struct {
	*baseDefaultFrame
}

// WriteableRequestStreamFrame is writeable RequestStream frame.
type WriteableRequestStreamFrame struct {
	baseWriteableFrame
	n        [4]byte
	metadata []byte
	data     []byte
}

// Validate returns error if frame is invalid.
func (r *RequestStreamFrame) Validate() error {
	l := r.body.Len()
	if l < minRequestStreamFrameLen {
		return errIncompleteFrame
	}
	if r.header.Flag().Check(core.FlagMetadata) && l < minRequestStreamFrameLen+3 {
		return errIncompleteFrame
	}
	return nil
}

// InitialRequestN returns initial request N.
func (r *RequestStreamFrame) InitialRequestN() uint32 {
	return binary.BigEndian.Uint32(r.body.Bytes())
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

// WriteTo writes frame to writer.
func (r WriteableRequestStreamFrame) WriteTo(w io.Writer) (n int64, err error) {
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
func (r WriteableRequestStreamFrame) Len() int {
	return 4 + CalcPayloadFrameSize(r.data, r.metadata)
}

// NewWriteableRequestStreamFrame creates a new WriteableRequestStreamFrame.
func NewWriteableRequestStreamFrame(id uint32, n uint32, data, metadata []byte, flag core.FrameFlag) core.WriteableFrame {
	if len(metadata) > 0 {
		flag |= core.FlagMetadata
	}
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], n)
	h := core.NewFrameHeader(id, core.FrameTypeRequestStream, flag)
	t := newBaseWriteableFrame(h)
	return &WriteableRequestStreamFrame{
		baseWriteableFrame: t,
		n:                  b,
		metadata:           metadata,
		data:               data,
	}
}

// NewRequestStreamFrame returns a new RequestStreamFrame.
func NewRequestStreamFrame(id uint32, n uint32, data, metadata []byte, flag core.FrameFlag) *RequestStreamFrame {
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
	return &RequestStreamFrame{
		newBaseDefaultFrame(core.NewFrameHeader(id, core.FrameTypeRequestStream, flag), b),
	}
}
