package framing

import (
	"io"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

// FireAndForgetFrame is FireAndForget frame.
type FireAndForgetFrame struct {
	*RawFrame
}

// WriteableFireAndForgetFrame is writeable FireAndForget frame.
type WriteableFireAndForgetFrame struct {
	*tinyFrame
	metadata []byte
	data     []byte
}

// WriteTo writes frame to writer.
func (f WriteableFireAndForgetFrame) WriteTo(w io.Writer) (n int64, err error) {
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

// Len returns length of frame.
func (f WriteableFireAndForgetFrame) Len() int {
	return CalcPayloadFrameSize(f.data, f.metadata)
}

// Validate returns error if frame is invalid.
func (f *FireAndForgetFrame) Validate() (err error) {
	if f.header.Flag().Check(core.FlagMetadata) && f.body.Len() < 3 {
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

// NewWriteableFireAndForgetFrame creates a new WriteableFireAndForgetFrame.
func NewWriteableFireAndForgetFrame(sid uint32, data, metadata []byte, flag core.FrameFlag) *WriteableFireAndForgetFrame {
	if len(metadata) > 0 {
		flag |= core.FlagMetadata
	}
	h := core.NewFrameHeader(sid, core.FrameTypeRequestFNF, flag)
	t := newTinyFrame(h)
	return &WriteableFireAndForgetFrame{
		tinyFrame: t,
		metadata:  metadata,
		data:      data,
	}
}

// NewFireAndForgetFrame returns a new FireAndForgetFrame.
func NewFireAndForgetFrame(sid uint32, data, metadata []byte, flag core.FrameFlag) *FireAndForgetFrame {
	bf := common.NewByteBuff()
	if len(metadata) > 0 {
		flag |= core.FlagMetadata
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
		NewRawFrame(core.NewFrameHeader(sid, core.FrameTypeRequestFNF, flag), bf),
	}
}
