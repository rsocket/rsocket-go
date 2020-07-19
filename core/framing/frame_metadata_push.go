package framing

import (
	"io"

	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/common"
)

var _metadataPushHeader = core.NewFrameHeader(0, core.FrameTypeMetadataPush, core.FlagMetadata)

// MetadataPushFrame is metadata push frame.
type MetadataPushFrame struct {
	*RawFrame
}
type WriteableMetadataPushFrame struct {
	*tinyFrame
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

func (m WriteableMetadataPushFrame) Len() int {
	return core.FrameHeaderLen + len(m.metadata)
}

// DataUTF8 returns data as UTF8 string.
func (m *MetadataPushFrame) DataUTF8() (data string) {
	return
}

func NewWriteableMetadataPushFrame(metadata []byte) *WriteableMetadataPushFrame {
	t := newTinyFrame(_metadataPushHeader)
	return &WriteableMetadataPushFrame{
		tinyFrame: t,
		metadata:  metadata,
	}
}

// NewMetadataPushFrame returns a new metadata push frame.
func NewMetadataPushFrame(metadata []byte) *MetadataPushFrame {
	bf := common.NewByteBuff()
	if _, err := bf.Write(metadata); err != nil {
		panic(err)
	}
	return &MetadataPushFrame{
		NewRawFrame(_metadataPushHeader, bf),
	}
}
