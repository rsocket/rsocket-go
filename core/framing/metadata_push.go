package framing

import (
	"github.com/rsocket/rsocket-go/core"
	"github.com/rsocket/rsocket-go/internal/bytebuffer"
	"github.com/rsocket/rsocket-go/internal/bytesconv"
)

// MetadataPushFrame is MetadataPush frame.
type MetadataPushFrame struct {
	*bufferedFrame
}

// NewMetadataPushFrame returns a new MetadataPushFrame.
func NewMetadataPushFrame(metadata []byte) *MetadataPushFrame {
	b := bytebuffer.BorrowByteBuff(core.FrameHeaderLen + len(metadata))

	if _, err := b.Write(_metadataPushHeader[:]); err != nil {
		bytebuffer.ReturnByteBuff(b)
		panic(err)
	}

	if _, err := b.Write(metadata); err != nil {
		bytebuffer.ReturnByteBuff(b)
		panic(err)
	}

	return &MetadataPushFrame{
		bufferedFrame: newBufferedFrame(b),
	}
}

// Validate returns error if frame is invalid.
func (m *MetadataPushFrame) Validate() (err error) {
	return
}

// Metadata returns metadata bytes.
func (m *MetadataPushFrame) Metadata() ([]byte, bool) {
	return m.Body(), true
}

// Data returns data bytes.
func (m *MetadataPushFrame) Data() []byte {
	return nil
}

// MetadataUTF8 returns metadata as UTF8 string.
func (m *MetadataPushFrame) MetadataUTF8() (metadata string, ok bool) {
	raw, ok := m.Metadata()
	if ok {
		metadata = bytesconv.BytesToString(raw)
	}
	return
}

// DataUTF8 returns data as UTF8 string.
func (m *MetadataPushFrame) DataUTF8() (data string) {
	return
}
