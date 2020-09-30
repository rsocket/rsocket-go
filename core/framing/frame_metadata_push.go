package framing

import (
	"io"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

var _metadataPushHeader = core.NewFrameHeader(0, core.FrameTypeMetadataPush, core.FlagMetadata)

// MetadataPushFrame is MetadataPush frame.
type MetadataPushFrame struct {
	*baseDefaultFrame
}

// WriteableMetadataPushFrame is writeable MetadataPush frame.
type WriteableMetadataPushFrame struct {
	baseWriteableFrame
	metadata []byte
}

// Validate returns error if frame is invalid.
func (m *MetadataPushFrame) Validate() (err error) {
	return
}

// Metadata returns metadata bytes.
func (m *MetadataPushFrame) Metadata() ([]byte, bool) {
	return m.body.Bytes(), true
}

// Data returns data bytes.
func (m *MetadataPushFrame) Data() []byte {
	return nil
}

// MetadataUTF8 returns metadata as UTF8 string.
func (m *MetadataPushFrame) MetadataUTF8() (metadata string, ok bool) {
	raw, ok := m.Metadata()
	if ok {
		metadata = string(raw)
	}
	return
}

// WriteTo writes frame to writer.
func (m WriteableMetadataPushFrame) WriteTo(w io.Writer) (n int64, err error) {
	var wrote int64
	wrote, err = m.header.WriteTo(w)
	if err != nil {
		return
	}
	n += wrote

	var v int
	v, err = w.Write(m.metadata)
	if err != nil {
		return
	}
	n += int64(v)
	return
}

// Len returns length of frame.
func (m WriteableMetadataPushFrame) Len() int {
	return core.FrameHeaderLen + len(m.metadata)
}

// DataUTF8 returns data as UTF8 string.
func (m *MetadataPushFrame) DataUTF8() (data string) {
	return
}

// NewWriteableMetadataPushFrame creates a new WriteableMetadataPushFrame.
func NewWriteableMetadataPushFrame(metadata []byte) *WriteableMetadataPushFrame {
	t := newBaseWriteableFrame(_metadataPushHeader)
	return &WriteableMetadataPushFrame{
		baseWriteableFrame: t,
		metadata:           metadata,
	}
}

// NewMetadataPushFrame returns a new MetadataPushFrame.
func NewMetadataPushFrame(metadata []byte) *MetadataPushFrame {
	b := common.BorrowByteBuff()
	if _, err := b.Write(metadata); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}
	return &MetadataPushFrame{
		newBaseDefaultFrame(_metadataPushHeader, b),
	}
}
