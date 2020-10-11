package framing

import (
	"io"

	"github.com/rsocket/rsocket-go/core"
)

var _metadataPushHeader = core.NewFrameHeader(0, core.FrameTypeMetadataPush, core.FlagMetadata)

// WriteableMetadataPushFrame is writeable MetadataPush frame.
type WriteableMetadataPushFrame struct {
	writeableFrame
	metadata []byte
}

// NewWriteableMetadataPushFrame creates a new WriteableMetadataPushFrame.
func NewWriteableMetadataPushFrame(metadata []byte) *WriteableMetadataPushFrame {
	t := newWriteableFrame(_metadataPushHeader)
	return &WriteableMetadataPushFrame{
		writeableFrame: t,
		metadata:       metadata,
	}
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
