package framing

import (
	"io"

	"github.com/rsocket/rsocket-go/internal/common"
)

// FireAndForgetFrame is fire and forget frame.
type FireAndForgetFrame struct {
	*RawFrame
}

type FireAndForgetFrameSupport struct {
	*tinyFrame
	metadata []byte
	data     []byte
}

func (f FireAndForgetFrameSupport) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int64
	wrote, err = f.header.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote

	wrote, err = writePayload(w, f.data, f.metadata)
	if err != nil {
		return
	}
	n += wrote
	return
}

func (f FireAndForgetFrameSupport) Len() int {
	return CalcPayloadFrameSize(f.data, f.metadata)
}

// Validate returns error if frame is invalid.
func (f *FireAndForgetFrame) Validate() (err error) {
	if f.header.Flag().Check(FlagMetadata) && f.body.Len() < 3 {
		err = errIncompleteFrame
	}
	return
}

// Metadata returns metadata bytes.
func (f *FireAndForgetFrame) Metadata() ([]byte, bool) {
	return f.trySliceMetadata(0)
}

// Data returns data bytes.
func (f *FireAndForgetFrame) Data() []byte {
	return f.trySliceData(0)
}

// MetadataUTF8 returns metadata as UTF8 string.
func (f *FireAndForgetFrame) MetadataUTF8() (metadata string, ok bool) {
	raw, ok := f.Metadata()
	if ok {
		metadata = string(raw)
	}
	return
}

// DataUTF8 returns data as UTF8 string.
func (f *FireAndForgetFrame) DataUTF8() string {
	return string(f.Data())
}

func NewFireAndForgetFrameSupport(sid uint32, data, metadata []byte, flag FrameFlag) *FireAndForgetFrameSupport {
	if len(metadata) > 0 {
		flag |= FlagMetadata
	}
	h := NewFrameHeader(sid, FrameTypeRequestFNF, flag)
	t := newTinyFrame(h)
	return &FireAndForgetFrameSupport{
		tinyFrame: t,
		metadata:  metadata,
		data:      data,
	}
}

// NewFireAndForgetFrame returns a new fire and forget frame.
func NewFireAndForgetFrame(sid uint32, data, metadata []byte, flag FrameFlag) *FireAndForgetFrame {
	bf := common.NewByteBuff()
	if len(metadata) > 0 {
		flag |= FlagMetadata
		if err := bf.WriteUint24(len(metadata)); err != nil {
			panic(err)
		}
		if _, err := bf.Write(metadata); err != nil {
			panic(err)
		}
	}
	if _, err := bf.Write(data); err != nil {
		panic(err)
	}
	return &FireAndForgetFrame{
		NewRawFrame(NewFrameHeader(sid, FrameTypeRequestFNF, flag), bf),
	}
}
