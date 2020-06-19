package framing

import (
	"encoding/binary"
	"io"

	"github.com/rsocket/rsocket-go/internal/common"
)

const (
	initReqLen                = 4
	minRequestChannelFrameLen = initReqLen
)

// RequestChannelFrame is frame for RequestChannel.
type RequestChannelFrame struct {
	*RawFrame
}

type RequestChannelFrameSupport struct {
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
	if r.header.Flag().Check(FlagMetadata) && l < minRequestChannelFrameLen+3 {
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

func (r RequestChannelFrameSupport) WriteTo(w io.Writer) (n int64, err error) {
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

func (r RequestChannelFrameSupport) Len() int {
	return CalcPayloadFrameSize(r.data, r.metadata) + 4
}

func NewRequestChannelFrameSupport(sid uint32, n uint32, data, metadata []byte, flag FrameFlag) *RequestChannelFrameSupport {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], n)
	if len(metadata) > 0 {
		flag |= FlagMetadata
	}
	h := NewFrameHeader(sid, FrameTypeRequestChannel, flag)
	t := newTinyFrame(h)
	return &RequestChannelFrameSupport{
		tinyFrame: t,
		n:         b,
		metadata:  metadata,
		data:      data,
	}
}

// NewRequestChannelFrame returns a new RequestChannel frame.
func NewRequestChannelFrame(sid uint32, n uint32, data, metadata []byte, flag FrameFlag) *RequestChannelFrame {
	bf := common.NewByteBuff()
	var b4 [4]byte
	binary.BigEndian.PutUint32(b4[:], n)
	if _, err := bf.Write(b4[:]); err != nil {
		panic(err)
	}
	if len(metadata) > 0 {
		flag |= FlagMetadata
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
		NewRawFrame(NewFrameHeader(sid, FrameTypeRequestChannel, flag), bf),
	}
}
