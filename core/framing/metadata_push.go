package framing

import (
	"github.com/rsocket/rsocket-go/internal/bytesconv"
	"github.com/rsocket/rsocket-go/internal/common"
)

// MetadataPushFrame is MetadataPush frame.
type MetadataPushFrame struct {
	*bufferedFrame
}

// NewMetadataPushFrame returns a new MetadataPushFrame.
func NewMetadataPushFrame(metadata []byte) *MetadataPushFrame {
	b := common.BorrowByteBuff()

	if _, err := b.Write(_metadataPushHeader[:]); err != nil {
		common.ReturnByteBuff(b)
		panic(err)
	}

	if _, err := b.Write(metadata); err != nil {
		common.ReturnByteBuff(b)
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
